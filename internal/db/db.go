package db

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/shengmingboai/octopus/internal/db/migrate"
	"github.com/shengmingboai/octopus/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

func InitDB(dbType, dsn string, debug bool) (err error) {
	conn, err := open(dbType, dsn, debug)
	if err == nil {
		db = conn
	}
	return err
}

// open 只在迁移全部完成后交出连接；初始化失败时关闭已分配的连接池。
func open(dbType, dsn string, debug bool) (_ *gorm.DB, err error) {
	var conn *gorm.DB
	gormConfig := gorm.Config{Logger: logger.Discard}
	if debug {
		gormConfig.Logger = logger.Default.LogMode(logger.Info)
	}

	switch dbType {
	case "sqlite":
		conn, err = initSQLite(dsn, &gormConfig)
	case "mysql":
		conn, err = initMySQL(dsn, &gormConfig)
	case "postgres", "postgresql":
		conn, err = initPostgres(dsn, &gormConfig)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}

	if err != nil {
		return nil, err
	}

	sqlDB, err := conn.DB()
	if err != nil {
		return nil, err
	}
	ready := false
	defer func() {
		if !ready {
			_ = sqlDB.Close()
		}
	}()

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	if err := migrate.BeforeAutoMigrate(conn); err != nil {
		return nil, err
	}
	if err := conn.AutoMigrate(
		&model.User{},
		&model.Channel{},
		&model.ChannelKey{},
		&model.ChannelModel{},
		&model.ChannelGrant{},
		&model.Group{},
		&model.GroupItem{},
		&model.LLMInfo{},
		&model.APIKey{},
		&model.Setting{},
		&model.StatsTotal{},
		&model.StatsDaily{},
		&model.StatsHourly{},
		&model.StatsAPIKey{},
		&migrate.MigrationRecord{},
	); err != nil {
		return nil, err
	}
	if err := migrate.AfterAutoMigrate(conn); err != nil {
		return nil, err
	}
	// Postgres: schema changes during migrations can invalidate cached prepared plans
	// (e.g. "cached plan must not change result type"). Clear them.
	if conn.Dialector != nil && conn.Dialector.Name() == "postgres" {
		conn.Exec("DEALLOCATE ALL")
		conn.Exec("DISCARD ALL")
	}
	ready = true
	return conn, nil
}

// initSQLite 使用指定文件路径初始化 SQLite，并为每个连接应用运行参数。
func initSQLite(path string, config *gorm.Config) (*gorm.DB, error) {
	params := url.Values{}
	params.Add("_pragma", "journal_mode(WAL)")
	params.Add("_pragma", "synchronous(NORMAL)")
	params.Add("_pragma", "cache_size(10000)")
	params.Add("_pragma", "busy_timeout(5000)")
	params.Add("_pragma", "foreign_keys(ON)")
	params.Add("_pragma", "auto_vacuum(INCREMENTAL)")
	params.Add("_pragma", "mmap_size(268435456)")
	params.Add("_pragma", "locking_mode(NORMAL)")
	return gorm.Open(sqlite.Open(path+"?"+params.Encode()), config)
}

func initMySQL(dsn string, config *gorm.Config) (*gorm.DB, error) {
	// DSN 格式: user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local
	if !strings.Contains(dsn, "?") {
		dsn += "?charset=utf8mb4&parseTime=True&loc=Local"
	}
	return gorm.Open(mysql.Open(dsn), config)
}

func initPostgres(dsn string, config *gorm.Config) (*gorm.DB, error) {
	// DSN 格式: host=localhost user=postgres password=xxx dbname=octopus port=5432 sslmode=disable
	return gorm.Open(postgres.Open(dsn), config)
}

func Close() error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func GetDB() *gorm.DB {
	return db
}
