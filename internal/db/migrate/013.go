package migrate

import (
	"fmt"

	"github.com/shengmingboai/octopus/internal/model"
	"gorm.io/gorm"
)

func init() {
	RegisterAfterAutoMigration(Migration{
		Version: 13,
		Up:      migrateChannelModelSource,
	})
}

// migrateChannelModelSource 把自动拉取功能启用前的既有渠道模型标记为自动来源。
// source 列由 AutoMigrate 按 manual 默认补齐, 但这些行都产生于自动拉取功能之外,
// 只有标成 auto 才会被同步任务按上游列表禁用与恢复, 人工后加的模型写入时即标 manual 不受影响。
func migrateChannelModelSource(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if !db.Migrator().HasTable("channel_models") || !hasPhysicalColumn(db, "channel_models", "source") {
		return nil
	}
	if err := db.Model(&model.ChannelModel{}).
		Where("source = ?", string(model.ChannelModelSourceManual)).
		Update("source", string(model.ChannelModelSourceAuto)).Error; err != nil {
		return fmt.Errorf("failed to mark channel model source: %w", err)
	}
	return nil
}
