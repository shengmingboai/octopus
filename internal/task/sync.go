package task

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/charmbracelet/log"

	"github.com/shengmingboai/octopus/internal/helper"
	"github.com/shengmingboai/octopus/internal/model"
	"github.com/shengmingboai/octopus/internal/op"
)

var (
	syncModelsMu         sync.Mutex   // 保证同一时间只有一个模型同步任务运行。
	lastSyncModelsTimeMu sync.RWMutex // 最近同步时间的读写锁。
	lastSyncModelsTime   time.Time    // 最近一次模型同步任务结束时间, 零值表示任务尚未跑过一轮。
)

// SyncModelsTask 自动拉取启用 AutoSync 的渠道的上游模型列表:
// 新模型按探测到的协议端点建授权, 上游不再提供的模型禁用而非删除, 恢复提供的模型重新启用。
// 人工来源的模型不参与增删改, 协议一旦配置好就不会被后续自动拉取改写。
// 返回本次同步遇到的首个错误, 供手动触发时在界面上提示。
func SyncModelsTask() error {
	if !syncModelsMu.TryLock() {
		return fmt.Errorf("model sync already running")
	}
	defer syncModelsMu.Unlock()

	log.Debugf("sync models task started")
	startTime := time.Now()
	defer func() {
		log.Debugf("sync models task finished, sync time: %s", time.Since(startTime))
	}()
	defer func() {
		lastSyncModelsTimeMu.Lock()
		lastSyncModelsTime = time.Now()
		lastSyncModelsTimeMu.Unlock()
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	var syncErr error
	for _, channelID := range op.ChannelIDs() {
		detail, err := op.ChannelDetailGet(channelID)
		if err != nil {
			log.Warnf("failed to load channel %d for sync: %v", channelID, err)
			continue
		}
		// 渠道自身被禁用时不拉取: 它不参与转发, 拉取也无法让模型重新可用;
		// AutoSync 为假的渠道模型全部人工维护, 与同步无关。
		if !detail.Enabled || !detail.AutoSync {
			continue
		}
		addedModels, regroup, err := syncChannelModelList(&detail, ctx)
		if err != nil {
			log.Warnf("failed to sync models for channel %s: %v", detail.Name, err)
			if syncErr == nil {
				syncErr = fmt.Errorf("failed to sync models for channel %s: %w", detail.Name, err)
			}
			continue
		}
		if len(addedModels) > 0 {
			if err := helper.LLMPricesAdd(addedModels, ctx); err != nil {
				log.Warnf("failed to add model prices for channel %s: %v", detail.Name, err)
			}
		}
		// 自动分组只在本轮新增或恢复模型时触发: 新增的模型首次出现, 恢复的模型重新启用后也要回到分组。
		if regroup {
			if err := op.AutoGroupChannel(&detail, ctx); err != nil {
				log.Warnf("failed to auto group channel %s: %v", detail.Name, err)
			}
		}
	}
	if err := op.LLMCleanupGhosts(ctx); err != nil {
		log.Errorf("failed to clean ghost model prices: %v", err)
		if syncErr == nil {
			syncErr = fmt.Errorf("failed to clean ghost model prices: %w", err)
		}
	}
	return syncErr
}

// syncChannelModelList 同步单个渠道的模型列表, 返回本次新增的模型与是否需要重新分组; 列表没有任何变化时返回空且不写库。
// 每条启用的凭据都拉一遍: 各凭据在上游被授权的模型不同, 新模型要在每条拉到它的凭据上都建授权。
// 拉取失败的凭据与拉通但为空的凭据不参与基准, 避免把其他凭据仍提供的模型误判为缺失而禁用。
func syncChannelModelList(detail *model.ChannelDetail, ctx context.Context) ([]model.ChannelModelConfig, bool, error) {
	var fetched []struct {
		keyName string
		models  []model.ChannelFetchModel
	}
	probeOK := false
	for _, channelKey := range detail.Keys {
		if !channelKey.Enabled || channelKey.Key == "" {
			continue
		}
		models, err := helper.FetchUpstreamModels(ctx, detail.ChannelConfig, channelKey.Key)
		if err != nil {
			log.Warnf("failed to fetch models for channel %s with key %s: %v", detail.Name, channelKey.Name, err)
			continue
		}
		probeOK = true
		if len(models) == 0 {
			// 这条凭据拉通了但一个模型都没给: 它在上游没有可见模型, 不参与基准。
			log.Warnf("channel %s fetched 0 models with key %s", detail.Name, channelKey.Name)
			continue
		}
		fetched = append(fetched, struct {
			keyName string
			models  []model.ChannelFetchModel
		}{keyName: channelKey.Name, models: models})
	}
	if !probeOK {
		return nil, false, fmt.Errorf("failed to fetch models for channel %s", detail.Name)
	}
	// 全部凭据都拉通但一个模型都没返回时不动本地列表: 瞬时异常返回空列表会把整条渠道的模型全部禁用。
	if len(fetched) == 0 {
		log.Warnf("channel %s fetched 0 models, skipped", detail.Name)
		return nil, false, nil
	}

	// 模型在任一凭据的非空结果中出现即视为上游仍提供; 协议位按凭据分别记录, 供新增模型逐凭据建授权。
	protocolsByKeyByModel := make(map[string]map[string]model.Protocol)
	for _, kf := range fetched {
		for _, fetchedModel := range kf.models {
			byKey := protocolsByKeyByModel[fetchedModel.Name]
			if byKey == nil {
				byKey = make(map[string]model.Protocol)
				protocolsByKeyByModel[fetchedModel.Name] = byKey
			}
			byKey[kf.keyName] |= fetchedModel.Protocols
		}
	}

	addedCount := 0
	disabledCount := 0
	restoredCount := 0
	models := make([]model.ChannelModelConfig, 0, len(detail.Models)+len(protocolsByKeyByModel))
	for _, channelModel := range detail.Models {
		if channelModel.Source == model.ChannelModelSourceManual {
			// 人工添加或人工接管的模型不受自动拉取影响: 上游缺失也保持人工给的启停状态,
			// 自定义模型与临时停用因此都不会被同步改写。
			models = append(models, channelModel)
			continue
		}
		if _, ok := protocolsByKeyByModel[channelModel.Name]; ok {
			// 上游仍在提供的模型重新启用: 此前被禁用的模型恢复, 其原有授权与分组成员原样保留。
			if !channelModel.Enabled {
				log.Infof("channel %s: model [%s] restored", detail.Name, channelModel.Name)
				channelModel.Enabled = true
				restoredCount++
			}
			models = append(models, channelModel)
			continue
		}
		// 上游不再提供的模型禁用而非删除: 授权与分组成员原样保留, 模型恢复时一并恢复。
		if channelModel.Enabled {
			log.Infof("channel %s: model [%s] disabled", detail.Name, channelModel.Name)
			channelModel.Enabled = false
			disabledCount++
		}
		models = append(models, channelModel)
	}

	addedModels := make([]model.ChannelModelConfig, 0, len(protocolsByKeyByModel))
	grants := make([]model.ChannelGrantConfig, 0, len(detail.Grants))
	grants = append(grants, detail.Grants...)
	existingNames := make(map[string]struct{}, len(detail.Models))
	for _, channelModel := range detail.Models {
		existingNames[channelModel.Name] = struct{}{}
	}
	// 按名称定序处理新模型, 新增授权与日志的顺序才不随 map 遍历变化;
	// 已有模型的授权保持原样, 不在此列: 人工配置好的协议不会被后续自动拉取改写。
	addedNames := make([]string, 0, len(protocolsByKeyByModel))
	for name := range protocolsByKeyByModel {
		if _, ok := existingNames[name]; !ok {
			addedNames = append(addedNames, name)
		}
	}
	sort.Strings(addedNames)
	for _, name := range addedNames {
		// 新模型按各凭据探测到的协议端点勾选, 每条拉到它的凭据都建一份授权。
		log.Infof("channel %s: model [%s] added", detail.Name, name)
		addedModels = append(addedModels, model.ChannelModelConfig{Name: name, Source: model.ChannelModelSourceAuto, Enabled: true})
		addedCount++
		for keyName, protocols := range protocolsByKeyByModel[name] {
			grants = append(grants, model.ChannelGrantConfig{ModelName: name, KeyName: keyName, Protocols: protocols})
		}
	}
	if addedCount == 0 && disabledCount == 0 && restoredCount == 0 {
		return nil, false, nil
	}
	// 新增模型并入提交集合: 授权按名称引用两侧, 两侧必须先在本事务内落库。
	models = append(models, addedModels...)
	if _, err := op.ChannelUpdate(&model.ChannelDetail{
		ID:            detail.ID,
		ChannelConfig: detail.ChannelConfig,
		Keys:          detail.Keys,
		Models:        models,
		Grants:        grants,
	}, ctx); err != nil {
		return nil, false, fmt.Errorf("failed to update channel %d models: %w", detail.ID, err)
	}
	if addedCount > 0 {
		log.Infof("channel %s: %d model(s) added", detail.Name, addedCount)
	}
	// 新增与恢复都需要重新分组: 新增的模型首次进入分组, 恢复的模型此前被禁用、重新启用后回到分组。
	return addedModels, addedCount > 0 || restoredCount > 0, nil
}

// GetLastSyncModelsTime 返回最近一次模型同步任务结束时间。
func GetLastSyncModelsTime() time.Time {
	lastSyncModelsTimeMu.RLock()
	defer lastSyncModelsTimeMu.RUnlock()
	return lastSyncModelsTime
}
