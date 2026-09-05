package task

import (
	"context"
	"time"

	"github.com/charmbracelet/log"
	"github.com/shengmingboai/octopus/internal/model"
	"github.com/shengmingboai/octopus/internal/op"
	"github.com/shengmingboai/octopus/internal/price"
	"github.com/shengmingboai/octopus/internal/probe"
)

func Init() error {
	// 注册价格更新任务
	Register(string(model.SettingKeyModelInfoUpdateInterval), 0, true, func(ctx context.Context) {
		if err := price.UpdateLLMPrice(ctx); err != nil {
			log.Warnf("failed to update price info: %v", err)
		}
	})

	// 注册统计保存任务
	Register(string(model.SettingKeyStatsSaveInterval), 0, false, op.StatsSaveDBTask)

	// 注册渠道模型同步任务: 逐个同步开启了自动同步的渠道, 单个渠道失败不影响其余渠道。
	Register(string(model.SettingKeySyncModelsInterval), 0, true, func(ctx context.Context) {
		summary, failed, err := probe.SyncAllChannels(ctx)
		if err != nil {
			log.Warnf("failed to sync channel models: %v", err)
		}
		if summary.AddedModels > 0 || summary.MissingGrants > 0 || summary.RestoredGrants > 0 {
			log.Infof("channel models synced: added %d, missing %d, restored %d, failed channels %d",
				summary.AddedModels, summary.MissingGrants, summary.RestoredGrants, failed)
		}
	})
	return RefreshIntervals()
}

// RefreshIntervals 从持久化设置的缓存读取最新间隔，启动、编辑和导入设置共用此入口。
func RefreshIntervals() error {
	for _, setting := range model.DefaultSettings() {
		unit := setting.Key.IntervalUnit()
		if unit == 0 {
			continue
		}
		value, err := op.SettingGetString(setting.Key)
		if err != nil {
			return err
		}
		setting.Value = value
		if err := setting.Validate(); err != nil {
			return err
		}
		interval, err := op.SettingGetInt(setting.Key)
		if err != nil {
			return err
		}
		Update(string(setting.Key), time.Duration(interval)*unit)
	}
	return nil
}
