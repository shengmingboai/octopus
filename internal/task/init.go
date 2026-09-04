package task

import (
	"context"
	"time"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/price"
	"github.com/bestruirui/octopus/internal/probe"
	"github.com/charmbracelet/log"
)

const (
	TaskPriceUpdate = "price_update"
	TaskStatsSave   = "stats_save"
	TaskCleanLLM    = "clean_llm"
)

func Init() {
	priceUpdateIntervalHours, err := op.SettingGetInt(model.SettingKeyModelInfoUpdateInterval)
	if err != nil {
		log.Errorf("failed to get model info update interval: %v", err)
		return
	}
	priceUpdateInterval := time.Duration(priceUpdateIntervalHours) * time.Hour
	// 注册价格更新任务
	Register(string(model.SettingKeyModelInfoUpdateInterval), priceUpdateInterval, true, func() {
		if err := price.UpdateLLMPrice(context.Background()); err != nil {
			log.Warnf("failed to update price info: %v", err)
		}
	})

	// 注册统计保存任务
	statsSaveIntervalMinutes, err := op.SettingGetInt(model.SettingKeyStatsSaveInterval)
	if err != nil {
		log.Warnf("failed to get stats save interval: %v", err)
		return
	}
	statsSaveInterval := time.Duration(statsSaveIntervalMinutes) * time.Minute
	Register(TaskStatsSave, statsSaveInterval, false, op.StatsSaveDBTask)

	// 注册渠道模型同步任务: 逐个同步开启了自动同步的渠道, 单个渠道失败不影响其余渠道。
	syncIntervalHours, err := op.SettingGetInt(model.SettingKeySyncModelsInterval)
	if err != nil {
		log.Warnf("failed to get sync models interval: %v", err)
		return
	}
	syncInterval := time.Duration(syncIntervalHours) * time.Hour
	Register(string(model.SettingKeySyncModelsInterval), syncInterval, true, func() {
		summary, failed, err := probe.SyncAllChannels(context.Background())
		if err != nil {
			log.Warnf("failed to sync channel models: %v", err)
		}
		if summary.AddedModels > 0 || summary.MissingGrants > 0 || summary.RestoredGrants > 0 {
			log.Infof("channel models synced: added %d, missing %d, restored %d, failed channels %d",
				summary.AddedModels, summary.MissingGrants, summary.RestoredGrants, failed)
		}
	})
}
