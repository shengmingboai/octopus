package op

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const dbDumpVersion = 1
const relayLogsBatchSize = 500

func DBExportAll(ctx context.Context, includeLogs, includeStats bool) (*model.DBDump, error) {
	conn := db.GetDB().WithContext(ctx)

	d := &model.DBDump{
		Version:      dbDumpVersion,
		ExportedAt:   time.Now().UTC(),
		IncludeLogs:  includeLogs,
		IncludeStats: includeStats,
	}

	if err := conn.Find(&d.Channels).Error; err != nil {
		return nil, fmt.Errorf("export channels: %w", err)
	}
	if err := conn.Find(&d.ChannelKeys).Error; err != nil {
		return nil, fmt.Errorf("export channel_keys: %w", err)
	}
	if err := conn.Find(&d.Groups).Error; err != nil {
		return nil, fmt.Errorf("export groups: %w", err)
	}
	if err := conn.Find(&d.GroupItems).Error; err != nil {
		return nil, fmt.Errorf("export group_items: %w", err)
	}
	if err := conn.Find(&d.LLMInfos).Error; err != nil {
		return nil, fmt.Errorf("export llm_infos: %w", err)
	}
	if err := conn.Find(&d.APIKeys).Error; err != nil {
		return nil, fmt.Errorf("export api_keys: %w", err)
	}
	if err := conn.Find(&d.Settings).Error; err != nil {
		return nil, fmt.Errorf("export settings: %w", err)
	}

	if includeStats {
		if err := conn.Find(&d.StatsTotal).Error; err != nil {
			return nil, fmt.Errorf("export stats_total: %w", err)
		}
		if err := conn.Find(&d.StatsDaily).Error; err != nil {
			return nil, fmt.Errorf("export stats_daily: %w", err)
		}
		if err := conn.Find(&d.StatsHourly).Error; err != nil {
			return nil, fmt.Errorf("export stats_hourly: %w", err)
		}
		if err := conn.Find(&d.StatsModel).Error; err != nil {
			return nil, fmt.Errorf("export stats_model: %w", err)
		}
		if err := conn.Find(&d.StatsChannel).Error; err != nil {
			return nil, fmt.Errorf("export stats_channel: %w", err)
		}
		if err := conn.Find(&d.StatsAPIKey).Error; err != nil {
			return nil, fmt.Errorf("export stats_api_key: %w", err)
		}
	}

	if includeLogs {
		if err := conn.Find(&d.RelayLogs).Error; err != nil {
			return nil, fmt.Errorf("export relay_logs: %w", err)
		}
	}

	return d, nil
}

// DBExportToFile 流式导出数据库到文件，支持分批读取 relay_logs 和实时进度推送
func DBExportToFile(ctx context.Context, taskID, filePath string, includeLogs, includeStats bool) error {
	conn := db.GetDB().WithContext(ctx)

	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)

	totalTables := 7
	if includeStats {
		totalTables += 6
	}
	if includeLogs {
		totalTables += 1
	}

	currentTable := 0
	sendProgress := func(tableName string, rows int64) {
		currentTable++
		notifyExportSubscribers(taskID, model.ExportProgress{
			Table:   tableName,
			Current: currentTable,
			Total:   totalTables,
			Rows:    rows,
			Status:  model.ExportStatusRunning,
		})
	}

	// 写入 JSON 头部
	if _, err := fmt.Fprintf(f, `{"version":%d,"exported_at":"%s","include_logs":%t,"include_stats":%t`,
		dbDumpVersion, time.Now().UTC().Format(time.RFC3339), includeLogs, includeStats); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	type tableEntry struct {
		name string
		fn   func() (any, int64, error)
	}

	tables := []tableEntry{
		{"channels", exportSmallTable[[]model.Channel](conn)},
		{"channel_keys", exportSmallTable[[]model.ChannelKey](conn)},
		{"groups", exportSmallTable[[]model.Group](conn)},
		{"group_items", exportSmallTable[[]model.GroupItem](conn)},
		{"llm_infos", exportSmallTable[[]model.LLMInfo](conn)},
		{"api_keys", exportSmallTable[[]model.APIKey](conn)},
		{"settings", exportSmallTable[[]model.Setting](conn)},
	}

	if includeStats {
		tables = append(tables,
			tableEntry{"stats_total", exportSmallTable[[]model.StatsTotal](conn)},
			tableEntry{"stats_daily", exportSmallTable[[]model.StatsDaily](conn)},
			tableEntry{"stats_hourly", exportSmallTable[[]model.StatsHourly](conn)},
			tableEntry{"stats_model", exportSmallTable[[]model.StatsModel](conn)},
			tableEntry{"stats_channel", exportSmallTable[[]model.StatsChannel](conn)},
			tableEntry{"stats_api_key", exportSmallTable[[]model.StatsAPIKey](conn)},
		)
	}

	for _, t := range tables {
		if err := ctx.Err(); err != nil {
			return err
		}
		data, rows, err := t.fn()
		if err != nil {
			return fmt.Errorf("export %s: %w", t.name, err)
		}
		if _, err := fmt.Fprintf(f, `,"%s":`, t.name); err != nil {
			return fmt.Errorf("write %s key: %w", t.name, err)
		}
		if err := enc.Encode(data); err != nil {
			return fmt.Errorf("encode %s: %w", t.name, err)
		}
		sendProgress(t.name, rows)
	}

	if includeLogs {
		if err := exportRelayLogsToFile(ctx, taskID, f, enc, conn, currentTable+1, totalTables); err != nil {
			return err
		}
		currentTable++
	}

	if _, err := fmt.Fprint(f, `}`); err != nil {
		return fmt.Errorf("write footer: %w", err)
	}

	return f.Sync()
}

func exportSmallTable[T any](conn *gorm.DB) func() (any, int64, error) {
	return func() (any, int64, error) {
		var data T
		result := conn.Find(&data)
		if result.Error != nil {
			return nil, 0, result.Error
		}
		return data, result.RowsAffected, nil
	}
}

// exportRelayLogsToFile 分批游标查询 relay_logs 并流式写入文件
// tableIndex 为 relay_logs 在总表序列中的序号（从 1 开始）
func exportRelayLogsToFile(ctx context.Context, taskID string, f *os.File, enc *json.Encoder, conn *gorm.DB, tableIndex, totalTables int) error {
	var totalCount int64
	if err := conn.Model(&model.RelayLog{}).Count(&totalCount).Error; err != nil {
		return fmt.Errorf("count relay_logs: %w", err)
	}

	if _, err := fmt.Fprint(f, `,"relay_logs":[`); err != nil {
		return fmt.Errorf("write relay_logs key: %w", err)
	}

	if totalCount == 0 {
		notifyExportSubscribers(taskID, model.ExportProgress{
			Table:   "relay_logs",
			Current: tableIndex,
			Total:   totalTables,
			Rows:    0,
			Status:  model.ExportStatusRunning,
		})
		_, err := fmt.Fprint(f, `]`)
		return err
	}

	var lastID int64
	var written int64
	first := true

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		var batch []model.RelayLog
		result := conn.Where("id > ?", lastID).Order("id ASC").Limit(relayLogsBatchSize).Find(&batch)
		if result.Error != nil {
			return fmt.Errorf("query relay_logs batch: %w", result.Error)
		}
		if len(batch) == 0 {
			break
		}

		lastID = batch[len(batch)-1].ID
		written += int64(len(batch))

		for _, logEntry := range batch {
			if !first {
				if _, err := fmt.Fprint(f, `,`); err != nil {
					return err
				}
			}
			first = false
			if err := enc.Encode(logEntry); err != nil {
				return fmt.Errorf("encode relay_log: %w", err)
			}
		}

		notifyExportSubscribers(taskID, model.ExportProgress{
			Table:   "relay_logs",
			Current: tableIndex,
			Total:   totalTables,
			Rows:    written,
			Status:  model.ExportStatusRunning,
		})

		if len(batch) < relayLogsBatchSize {
			break
		}
	}

	notifyExportSubscribers(taskID, model.ExportProgress{
		Table:   "relay_logs",
		Current: tableIndex,
		Total:   totalTables,
		Rows:    written,
		Status:  model.ExportStatusRunning,
	})

	_, err := fmt.Fprint(f, `]`)
	return err
}

func DBImportIncremental(ctx context.Context, dump *model.DBDump) (*model.DBImportResult, error) {
	if dump == nil {
		return nil, fmt.Errorf("empty dump")
	}

	if dump.Version != 0 && dump.Version != dbDumpVersion {
		return nil, fmt.Errorf("unsupported dump version: %d", dump.Version)
	}

	conn := db.GetDB().WithContext(ctx)
	res := &model.DBImportResult{RowsAffected: map[string]int64{}}

	err := conn.Transaction(func(tx *gorm.DB) error {
		// base tables
		if n, err := createDoNothing(tx, dump.Channels); err != nil {
			return fmt.Errorf("import channels: %w", err)
		} else {
			res.RowsAffected["channels"] = n
		}
		if n, err := createDoNothing(tx, dump.ChannelKeys); err != nil {
			return fmt.Errorf("import channel_keys: %w", err)
		} else {
			res.RowsAffected["channel_keys"] = n
		}
		if n, err := createDoNothing(tx, dump.Groups); err != nil {
			return fmt.Errorf("import groups: %w", err)
		} else {
			res.RowsAffected["groups"] = n
		}
		if n, err := createDoNothing(tx, dump.GroupItems); err != nil {
			return fmt.Errorf("import group_items: %w", err)
		} else {
			res.RowsAffected["group_items"] = n
		}
		if n, err := createUpsertAll(tx, dump.LLMInfos, []clause.Column{{Name: "name"}}); err != nil {
			return fmt.Errorf("import llm_infos: %w", err)
		} else {
			res.RowsAffected["llm_infos"] = n
		}
		if n, err := createDoNothing(tx, dump.APIKeys); err != nil {
			return fmt.Errorf("import api_keys: %w", err)
		} else {
			res.RowsAffected["api_keys"] = n
		}
		if n, err := createUpsertSettings(tx, dump.Settings); err != nil {
			return fmt.Errorf("import settings: %w", err)
		} else {
			res.RowsAffected["settings"] = n
		}

		if dump.IncludeStats {
			if n, err := createUpsertAll(tx, dump.StatsTotal, []clause.Column{{Name: "id"}}); err != nil {
				return fmt.Errorf("import stats_total: %w", err)
			} else {
				res.RowsAffected["stats_total"] = n
			}
			if n, err := createUpsertAll(tx, dump.StatsDaily, []clause.Column{{Name: "date"}}); err != nil {
				return fmt.Errorf("import stats_daily: %w", err)
			} else {
				res.RowsAffected["stats_daily"] = n
			}
			if n, err := createUpsertAll(tx, dump.StatsHourly, []clause.Column{{Name: "hour"}}); err != nil {
				return fmt.Errorf("import stats_hourly: %w", err)
			} else {
				res.RowsAffected["stats_hourly"] = n
			}
			if n, err := createUpsertAll(tx, dump.StatsModel, []clause.Column{{Name: "id"}}); err != nil {
				return fmt.Errorf("import stats_model: %w", err)
			} else {
				res.RowsAffected["stats_model"] = n
			}
			if n, err := createUpsertAll(tx, dump.StatsChannel, []clause.Column{{Name: "channel_id"}}); err != nil {
				return fmt.Errorf("import stats_channel: %w", err)
			} else {
				res.RowsAffected["stats_channel"] = n
			}
			if n, err := createUpsertAll(tx, dump.StatsAPIKey, []clause.Column{{Name: "api_key_id"}}); err != nil {
				return fmt.Errorf("import stats_api_key: %w", err)
			} else {
				res.RowsAffected["stats_api_key"] = n
			}
		}

		if dump.IncludeLogs {
			if n, err := createDoNothing(tx, dump.RelayLogs); err != nil {
				return fmt.Errorf("import relay_logs: %w", err)
			} else {
				res.RowsAffected["relay_logs"] = n
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func createDoNothing[T any](tx *gorm.DB, rows []T) (int64, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows)
	return result.RowsAffected, result.Error
}

func createUpsertAll[T any](tx *gorm.DB, rows []T, columns []clause.Column) (int64, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	result := tx.Clauses(clause.OnConflict{
		Columns:   columns,
		UpdateAll: true,
	}).Create(&rows)
	return result.RowsAffected, result.Error
}

func createUpsertSettings(tx *gorm.DB, rows []model.Setting) (int64, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	result := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value"}),
	}).Create(&rows)
	return result.RowsAffected, result.Error
}
