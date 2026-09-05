package migrate

import (
	"github.com/shengmingboai/octopus/internal/model"
	"gorm.io/gorm"
)

func init() {
	RegisterAfterAutoMigration(Migration{
		Version: 14,
		Up:      dropGroupRelayConfig,
	})
}

// dropGroupRelayConfig 删除分组的 relay_config 列。
// 路由策略已全局化到设置页, 且按约定不做配置值迁移: 升级后全部沿用全局默认值。
func dropGroupRelayConfig(db *gorm.DB) error {
	if db == nil {
		return gorm.ErrInvalidDB
	}
	if !db.Migrator().HasTable("groups") {
		return nil
	}
	return dropColumnIfExists(db, &model.Group{}, "groups", "relay_config")
}
