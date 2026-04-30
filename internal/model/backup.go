package model

import (
	"time"
)

// ExportTaskStatus 导出任务状态
type ExportTaskStatus string

const (
	ExportStatusRunning   ExportTaskStatus = "running"
	ExportStatusDone      ExportTaskStatus = "done"
	ExportStatusError     ExportTaskStatus = "error"
	ExportStatusCancelled ExportTaskStatus = "cancelled"
)

// ExportTask 异步导出任务
type ExportTask struct {
	ID        string           `json:"id"`
	Status    ExportTaskStatus `json:"status"`
	FilePath  string           `json:"-"`
	FileName  string           `json:"file_name"`
	CreatedAt time.Time        `json:"created_at"`
	Error     string           `json:"error,omitempty"`
}

// ExportProgress 导出进度事件（通过 SSE 推送到前端）
type ExportProgress struct {
	Table   string           `json:"table"`
	Current int              `json:"current"`
	Total   int              `json:"total"`
	Rows    int64            `json:"rows,omitempty"`
	Status  ExportTaskStatus `json:"status,omitempty"`
	Error   string           `json:"error,omitempty"`
}

// DBDump is a full-database JSON export format for Octopus.
// Import uses incremental semantics (insert new rows, and upsert on certain key-based tables).
type DBDump struct {
	Version      int       `json:"version"`
	ExportedAt   time.Time `json:"exported_at"`
	IncludeLogs  bool      `json:"include_logs"`
	IncludeStats bool      `json:"include_stats"`

	Channels    []Channel    `json:"channels,omitempty"`
	ChannelKeys []ChannelKey `json:"channel_keys,omitempty"`
	Groups      []Group      `json:"groups,omitempty"`
	GroupItems []GroupItem `json:"group_items,omitempty"`
	LLMInfos   []LLMInfo   `json:"llm_infos,omitempty"`
	APIKeys    []APIKey    `json:"api_keys,omitempty"`
	Settings   []Setting   `json:"settings,omitempty"`

	StatsTotal   []StatsTotal   `json:"stats_total,omitempty"`
	StatsDaily   []StatsDaily   `json:"stats_daily,omitempty"`
	StatsHourly  []StatsHourly  `json:"stats_hourly,omitempty"`
	StatsModel   []StatsModel   `json:"stats_model,omitempty"`
	StatsChannel []StatsChannel `json:"stats_channel,omitempty"`
	StatsAPIKey  []StatsAPIKey  `json:"stats_api_key,omitempty"`

	RelayLogs []RelayLog `json:"relay_logs,omitempty"`
}

type DBImportResult struct {
	// RowsAffected contains the rows affected for each table operation (insert/upsert depending on table).
	RowsAffected map[string]int64 `json:"rows_affected"`
}
