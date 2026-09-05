package migrate

import (
	"fmt"

	"github.com/shengmingboai/octopus/internal/model"
	"gorm.io/gorm"
)

func init() {
	RegisterAfterAutoMigration(Migration{
		Version: 7,
		Up:      migrateGroupRouting,
	})
}

// migrateGroupRouting 删除旧版分组的 retry_interval 列。
// 路由策略已全局化到设置页: 旧的分组级配置值不再迁移, 各分组直接沿用全局策略(缺失时按默认值兜底);
// mode 列由自动迁移补齐, 历史行缺值时按手动模式兜底。
func migrateGroupRouting(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if !db.Migrator().HasTable("groups") {
		return nil
	}
	if !hasPhysicalColumn(db, "groups", "retry_interval") {
		return nil
	}
	if hasPhysicalColumn(db, "groups", "mode") {
		if err := db.Table("groups").Where("mode = '' OR mode IS NULL").
			Update("mode", model.GroupModeManual).Error; err != nil {
			return fmt.Errorf("failed to default group mode: %w", err)
		}
	}
	return dropColumnIfExists(db, &model.Group{}, "groups", "retry_interval")
}
