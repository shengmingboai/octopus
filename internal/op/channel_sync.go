package op

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/shengmingboai/octopus/internal/db"
	"github.com/shengmingboai/octopus/internal/model"
	"gorm.io/gorm"
)

// AutoSyncChannelIDs 返回已开启自动同步且启用的渠道主键, 供定时任务遍历。
func AutoSyncChannelIDs() []int {
	ids := make([]int, 0, channelCache.Len())
	for _, channel := range channelCache.GetAll() {
		if channel.Enabled && channel.AutoSyncModels {
			ids = append(ids, channel.ID)
		}
	}
	return ids
}

// ChannelSyncApply 按探测结果同步渠道的模型与授权, 返回变化摘要。
// probeByKeyName 的键是凭据名称, 值是该凭据探测到的模型及其协议位; 只传入探测成功的凭据,
// 探测失败或结果为空的凭据不参与本轮同步, 其授权原样保留。
// 同步语义:
//   - 探测结果里有而库里没有的模型: 新建渠道模型, 并为每个探测到它的凭据按探测协议建授权;
//     手动删除的模型会由此回来, 渠道开启自动同步即表示模型集合交给同步管理。
//   - 已有的模型×凭据授权: 只翻转 Missing 标记, 协议位永不改写 —— 用户首次确认的协议永久生效,
//     模型上游消失时标记 Missing 保留在库(分组项与统计都不动), 恢复后按原协议原位置复活。
//   - 渠道开启自动入组时, 新引入的模型若与某分组名忽略大小写精确一致, 其新授权自动加为该分组成员,
//     排在现有成员之后; 从上游消失恢复的授权按同样规则重新入组, 已在该分组的成员不重复入组;
//     其余已有模型的授权仍由用户自行管理。
func ChannelSyncApply(channelID int, probeByKeyName map[string][]model.ChannelFetchModel, ctx context.Context) (*model.ChannelSyncResult, error) {
	channel, ok := channelCache.Get(channelID)
	if !ok {
		return nil, fmt.Errorf("channel not found")
	}
	result := &model.ChannelSyncResult{}
	// 待入组成员在同一事务内创建: 新授权的主键事务内才拿得到, 与模型和授权的写入保持原子。
	type pendingItem struct {
		groupID   int    // 目标分组主键。
		grantID   int    // 待入组的渠道授权主键。
		modelName string // 授权引用的模型名, 供入组前定序。
		keyName   string // 授权引用的凭据名, 供入组前定序。
	}
	pendingItems := make([]pendingItem, 0)
	if err := db.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 自动入组按分组名匹配模型名: 忽略大小写精确一致, 模型名带供应商前缀时取最后一个 / 之后的
		// 名索引比对, 各方言 LOWER 函数对非 ASCII 的行为差异由此规避。同形异名分组取主键最小的一个, 顺序稳定。
		groupIDByLower := make(map[string]int)
		matchGroup := func(modelName string) (int, bool) {
			if groupID, ok := groupIDByLower[strings.ToLower(modelName)]; ok {
				return groupID, true
			}
			if idx := strings.LastIndex(modelName, "/"); idx >= 0 {
				groupID, ok := groupIDByLower[strings.ToLower(modelName[idx+1:])]
				return groupID, ok
			}
			return 0, false
		}
		if channel.AutoGroup {
			groups := []model.Group{}
			if err := tx.Select("id, name").Find(&groups).Error; err != nil {
				return fmt.Errorf("failed to load groups: %w", err)
			}
			for _, group := range groups {
				lower := strings.ToLower(group.Name)
				if current, ok := groupIDByLower[lower]; !ok || group.ID < current {
					groupIDByLower[lower] = group.ID
				}
			}
		}

		channelModels := []model.ChannelModel{}
		if err := tx.Where("channel_id = ?", channelID).Find(&channelModels).Error; err != nil {
			return fmt.Errorf("failed to load channel models: %w", err)
		}
		modelIDByName := make(map[string]int, len(channelModels))
		modelNameByID := make(map[int]string, len(channelModels))
		modelIDs := make([]int, 0, len(channelModels))
		for _, channelModel := range channelModels {
			modelIDByName[channelModel.Name] = channelModel.ID
			modelNameByID[channelModel.ID] = channelModel.Name
			modelIDs = append(modelIDs, channelModel.ID)
		}

		channelKeys := []model.ChannelKey{}
		if err := tx.Where("channel_id = ?", channelID).Find(&channelKeys).Error; err != nil {
			return fmt.Errorf("failed to load channel keys: %w", err)
		}
		keyIDByName := make(map[string]int, len(channelKeys))
		for _, channelKey := range channelKeys {
			keyIDByName[channelKey.Name] = channelKey.ID
		}

		// 提交的凭据名必须仍是本渠道的凭据: 探测与落库之间凭据可能被删除, 此时本轮同步按失败返回。
		keyIDs := make([]int, 0, len(probeByKeyName))
		for keyName := range probeByKeyName {
			keyID, ok := keyIDByName[keyName]
			if !ok {
				return fmt.Errorf("channel key %q does not belong to channel %d", keyName, channelID)
			}
			keyIDs = append(keyIDs, keyID)
		}

		existingGrants := []model.ChannelGrant{}
		if len(modelIDs) > 0 {
			if err := tx.Where("channel_model_id IN ?", modelIDs).Find(&existingGrants).Error; err != nil {
				return fmt.Errorf("failed to load channel grants: %w", err)
			}
		}
		type grantKey struct {
			modelID int // 渠道模型主键。
			keyID   int // 渠道凭据主键。
		}
		grantByCombo := make(map[grantKey]model.ChannelGrant, len(existingGrants))
		for _, grant := range existingGrants {
			grantByCombo[grantKey{grant.ChannelModelID, grant.ChannelKeyID}] = grant
		}

		for keyName, probedModels := range probeByKeyName {
			keyID := keyIDByName[keyName]
			probedByName := make(map[string]model.Protocol, len(probedModels))
			for _, probed := range probedModels {
				probedByName[probed.Name] = probed.Protocols
			}

			for modelName, protocols := range probedByName {
				isNewModel := false
				modelID, ok := modelIDByName[modelName]
				if !ok {
					newModel := model.ChannelModel{ChannelID: channelID, Name: modelName}
					if err := tx.Create(&newModel).Error; err != nil {
						return fmt.Errorf("failed to create channel model: %w", err)
					}
					modelIDByName[modelName] = newModel.ID
					modelNameByID[newModel.ID] = modelName
					modelIDs = append(modelIDs, newModel.ID)
					modelID = newModel.ID
					result.AddedModels++
					result.AddedModelName = append(result.AddedModelName, modelName)
					isNewModel = true
				}
				grant, ok := grantByCombo[grantKey{modelID, keyID}]
				if !ok {
					// 新授权直接采用探测协议; 上线后协议位归用户所有, 后续同步不再改写。
					newGrant := model.ChannelGrant{
						ChannelModelID: modelID,
						ChannelKeyID:   keyID,
						Protocols:      protocols,
					}
					if err := tx.Create(&newGrant).Error; err != nil {
						return fmt.Errorf("failed to create channel grant: %w", err)
					}
					if isNewModel {
						if groupID, ok := matchGroup(modelName); ok {
							pendingItems = append(pendingItems, pendingItem{
								groupID:   groupID,
								grantID:   newGrant.ID,
								modelName: modelName,
								keyName:   keyName,
							})
						}
					}
					continue
				}
				if grant.Missing {
					// 模型在上游恢复: 按首次保存的协议原样复活, 协议位不动。
					if err := tx.Model(&model.ChannelGrant{}).Where("id = ?", grant.ID).
						Update("missing", false).Error; err != nil {
						return fmt.Errorf("failed to restore channel grant: %w", err)
					}
					result.RestoredGrants++
					// 复活的授权与新模型同规则自动入组: 上游恢复即重新成为成员, 排在现有成员之后,
					// 分组项被手动移除过的也由此回来; 已在该分组的不重复入组。
					if groupID, ok := matchGroup(modelName); ok {
						var count int64
						if err := tx.Model(&model.GroupItem{}).
							Where("group_id = ? AND channel_grant_id = ?", groupID, grant.ID).
							Count(&count).Error; err != nil {
							return fmt.Errorf("failed to load group item: %w", err)
						}
						if count == 0 {
							pendingItems = append(pendingItems, pendingItem{
								groupID:   groupID,
								grantID:   grant.ID,
								modelName: modelName,
								keyName:   keyName,
							})
						}
					}
				}
			}

			// 本凭据现有授权中探测结果里没有的模型: 标记为上游消失, 协议位与分组项原样保留。
			for combo, grant := range grantByCombo {
				if combo.keyID != keyID || grant.Missing {
					continue
				}
				if _, probed := probedByName[modelNameByID[combo.modelID]]; probed {
					continue
				}
				if err := tx.Model(&model.ChannelGrant{}).Where("id = ?", grant.ID).
					Update("missing", true).Error; err != nil {
					return fmt.Errorf("failed to mark channel grant missing: %w", err)
				}
				result.MissingGrants++
			}
		}

		// 待入组成员统一落库: 按分组, 模型名, 凭据名定序后排在各分组现有成员之后,
		// 缓存遍历顺序随机, 入组顺序须由此定稿, 才不会随每次同步抖动。
		if len(pendingItems) > 0 {
			sort.Slice(pendingItems, func(i, j int) bool {
				if pendingItems[i].groupID != pendingItems[j].groupID {
					return pendingItems[i].groupID < pendingItems[j].groupID
				}
				if pendingItems[i].modelName != pendingItems[j].modelName {
					return pendingItems[i].modelName < pendingItems[j].modelName
				}
				return pendingItems[i].keyName < pendingItems[j].keyName
			})
			nextPriority := make(map[int]int)
			for _, item := range pendingItems {
				if _, ok := nextPriority[item.groupID]; !ok {
					var maxPriority int
					if err := tx.Model(&model.GroupItem{}).Where("group_id = ?", item.groupID).
						Select("COALESCE(MAX(priority), 0)").Scan(&maxPriority).Error; err != nil {
						return fmt.Errorf("failed to load group max priority: %w", err)
					}
					nextPriority[item.groupID] = maxPriority + 1
				}
				newItem := model.GroupItem{
					GroupID:        item.groupID,
					ChannelGrantID: item.grantID,
					Priority:       nextPriority[item.groupID],
				}
				nextPriority[item.groupID]++
				if err := tx.Create(&newItem).Error; err != nil {
					return fmt.Errorf("failed to create group item: %w", err)
				}
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// 授权的 Missing 变化会改变可用路由集合与分组展示, 重载该渠道的子表缓存;
	// 有成员入组时分组缓存一并刷新, 新成员才能出现在分组页与转发选路里。
	if err := reloadChannelChildren(ctx, channelID); err != nil {
		return nil, err
	}
	if len(pendingItems) > 0 {
		if err := groupRefreshCache(ctx); err != nil {
			return nil, fmt.Errorf("failed to refresh groups: %w", err)
		}
	}
	return result, nil
}
