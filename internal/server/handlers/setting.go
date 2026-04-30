package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/server/middleware"
	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/bestruirui/octopus/internal/server/router"
	"github.com/bestruirui/octopus/internal/task"
	"github.com/gin-gonic/gin"
)

func init() {
	router.NewGroupRouter("/api/v1/setting").
		Use(middleware.Auth()).
		AddRoute(
			router.NewRoute("/list", http.MethodGet).
				Handle(getSettingList),
		).
		AddRoute(
			router.NewRoute("/set", http.MethodPost).
				Use(middleware.RequireJSON()).
				Handle(setSetting),
		).
		AddRoute(
			router.NewRoute("/export/start", http.MethodPost).
				Handle(exportStart),
		).
		AddRoute(
			router.NewRoute("/export/progress", http.MethodGet).
				Handle(exportProgressSSE),
		).
		AddRoute(
			router.NewRoute("/export/cancel", http.MethodPost).
				Handle(exportCancel),
		).
		AddRoute(
			router.NewRoute("/export/download", http.MethodGet).
				Handle(exportDownload),
		).
		AddRoute(
			router.NewRoute("/import", http.MethodPost).
				Handle(importDB),
		)
}

func getSettingList(c *gin.Context) {
	settings, err := op.SettingList(c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, settings)
}

func setSetting(c *gin.Context) {
	var setting model.Setting
	if err := c.ShouldBindJSON(&setting); err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := setting.Validate(); err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := op.SettingSetString(setting.Key, setting.Value); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	switch setting.Key {
	case model.SettingKeyModelInfoUpdateInterval:
		hours, err := strconv.Atoi(setting.Value)
		if err != nil {
			resp.Error(c, http.StatusBadRequest, err.Error())
			return
		}
		task.Update(string(setting.Key), time.Duration(hours)*time.Hour)
	case model.SettingKeySyncLLMInterval:
		hours, err := strconv.Atoi(setting.Value)
		if err != nil {
			resp.Error(c, http.StatusBadRequest, err.Error())
			return
		}
		task.Update(string(setting.Key), time.Duration(hours)*time.Hour)
	}
	resp.Success(c, setting)
}

type exportStartRequest struct {
	IncludeLogs  bool `json:"include_logs"`
	IncludeStats bool `json:"include_stats"`
}

func exportStart(c *gin.Context) {
	var req exportStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	taskID, err := op.ExportStart(req.IncludeLogs, req.IncludeStats)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	resp.Success(c, gin.H{"task_id": taskID})
}

func exportProgressSSE(c *gin.Context) {
	taskID := c.Query("task_id")
	if taskID == "" {
		resp.Error(c, http.StatusBadRequest, "task_id is required")
		return
	}

	task := op.ExportGetTask(taskID)
	if task == nil {
		resp.Error(c, http.StatusNotFound, "task not found")
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	if task.Status != model.ExportStatusRunning {
		data, _ := json.Marshal(model.ExportProgress{
			Status: task.Status,
			Error:  task.Error,
		})
		c.Writer.Write([]byte(fmt.Sprintf("data: %s\n\n", data)))
		c.Writer.Flush()
		return
	}

	// 先发一个连接确认事件，确保前端 ReadableStream 能立即读到数据
	c.Writer.Write([]byte("data: {\"status\":\"connecting\"}\n\n"))
	c.Writer.Flush()

	ch := op.ExportSubscribe(taskID)
	defer op.ExportUnsubscribe(taskID, ch)

	doneCh := op.ExportGetDoneCh(taskID)
	ctx := c.Request.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-doneCh:
			// 导出完成，发送最终事件（兜底，防止 channel 事件被丢弃）
			final := op.ExportGetFinalEvent(taskID)
			data, _ := json.Marshal(final)
			c.Writer.Write([]byte(fmt.Sprintf("data: %s\n\n", data)))
			c.Writer.Flush()
			return
		case progress, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(progress)
			if err != nil {
				continue
			}
			c.Writer.Write([]byte(fmt.Sprintf("data: %s\n\n", data)))
			c.Writer.Flush()

			if progress.Status != model.ExportStatusRunning {
				return
			}
		}
	}
}

func exportCancel(c *gin.Context) {
	var req struct {
		TaskID string `json:"task_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := op.ExportCancel(req.TaskID); err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	resp.Success(c, nil)
}

func exportDownload(c *gin.Context) {
	taskID := c.Query("task_id")
	if taskID == "" {
		resp.Error(c, http.StatusBadRequest, "task_id is required")
		return
	}

	task := op.ExportGetTask(taskID)
	if task == nil {
		resp.Error(c, http.StatusNotFound, "task not found")
		return
	}

	if task.Status != model.ExportStatusDone {
		resp.Error(c, http.StatusBadRequest, "export not completed, status: "+string(task.Status))
		return
	}

	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "attachment; filename=\""+task.FileName+"\"")
	c.File(task.FilePath)
}

func importDB(c *gin.Context) {
	var dump model.DBDump

	contentType := c.GetHeader("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		fh, err := c.FormFile("file")
		if err != nil {
			resp.Error(c, http.StatusBadRequest, "missing upload file field 'file'")
			return
		}
		f, err := fh.Open()
		if err != nil {
			resp.Error(c, http.StatusBadRequest, err.Error())
			return
		}
		defer f.Close()
		body, err := io.ReadAll(f)
		if err != nil {
			resp.Error(c, http.StatusBadRequest, err.Error())
			return
		}
		if err := decodeDBDump(body, &dump); err != nil {
			resp.Error(c, http.StatusBadRequest, err.Error())
			return
		}
	} else {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			resp.Error(c, http.StatusBadRequest, err.Error())
			return
		}
		if err := decodeDBDump(body, &dump); err != nil {
			resp.Error(c, http.StatusBadRequest, err.Error())
			return
		}
	}

	result, err := op.DBImportIncremental(c.Request.Context(), &dump)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	_ = op.InitCache()

	resp.Success(c, result)
}

func decodeDBDump(body []byte, dump *model.DBDump) error {
	if dump == nil {
		return json.Unmarshal(body, &struct{}{})
	}

	if err := json.Unmarshal(body, dump); err != nil {
		return err
	}

	if dump.Version == 0 &&
		len(dump.Channels) == 0 &&
		len(dump.Groups) == 0 &&
		len(dump.GroupItems) == 0 &&
		len(dump.Settings) == 0 &&
		len(dump.APIKeys) == 0 &&
		len(dump.LLMInfos) == 0 &&
		len(dump.RelayLogs) == 0 &&
		len(dump.StatsDaily) == 0 &&
		len(dump.StatsHourly) == 0 &&
		len(dump.StatsTotal) == 0 &&
		len(dump.StatsChannel) == 0 &&
		len(dump.StatsModel) == 0 &&
		len(dump.StatsAPIKey) == 0 {
		var wrapper struct {
			Code    int             `json:"code"`
			Message string          `json:"message"`
			Data    json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(body, &wrapper); err == nil && len(wrapper.Data) > 0 {
			return json.Unmarshal(wrapper.Data, dump)
		}
	}

	return nil
}
