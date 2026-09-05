package migrate

import (
	"fmt"

	"gorm.io/gorm"
)

func init() {
	RegisterAfterAutoMigration(Migration{
		Version: 13,
		Up:      migrateGroupItemsSequence,
	})
}

// migrateGroupItemsSequence 把 group_items 的自增序列推进到现有最大主键之后。
// 版本 8 与 11 的迁移整表重建该表并按原主键回填数据, PostgreSQL 的序列不会因显式主键插入而推进,
// 重建后的序列从 1 重新计数, 运行期插入新成员便与保留的旧主键冲突(23505)。已踩坑的库由此迁移自愈。
func migrateGroupItemsSequence(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	return resetGroupItemsSequence(db)
}

// resetGroupItemsSequence 将 group_items 的自增序列推进到现有最大主键之后, 幂等可重复执行。
// 只在 PostgreSQL 需要: SQLite 的整型主键自动取最大值加一, MySQL 插入显式主键时会自动推进计数。
func resetGroupItemsSequence(db *gorm.DB) error {
	if db.Dialector.Name() != "postgres" {
		return nil
	}
	if !db.Migrator().HasTable("group_items") {
		return nil
	}
	// 序列由 pg_get_serial_sequence 定位, 表存在但无序列时跳过; 置位后下一id为 max(id)+1, 空表则为 1。
	if err := db.Exec(`SELECT setval(pg_get_serial_sequence('group_items', 'id'), ` +
		`COALESCE((SELECT MAX(id) FROM group_items), 0) + 1, false) ` +
		`WHERE pg_get_serial_sequence('group_items', 'id') IS NOT NULL`).Error; err != nil {
		return fmt.Errorf("failed to reset group_items id sequence: %w", err)
	}
	return nil
}
