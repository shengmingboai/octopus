package op

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/utils/log"
)

const (
	exportCleanupInterval = 5 * time.Minute
	exportTaskTTL         = 30 * time.Minute
)

var (
	exportTasks         = make(map[string]*exportTaskState)
	exportTasksMu       sync.RWMutex
	exportSubscribers   = make(map[string][]chan model.ExportProgress)
	exportSubscribersMu sync.RWMutex
)

type exportTaskState struct {
	task       model.ExportTask
	cancel     context.CancelFunc
	doneCh     chan struct{} // closed when export finishes
	finalEvent model.ExportProgress
}

func init() {
	go exportCleanupLoop()
}

// ExportStart 创建异步导出任务，返回 taskID
func ExportStart(includeLogs, includeStats bool) (string, error) {
	taskID := generateID()

	tmpDir := os.TempDir()
	fileName := fmt.Sprintf("octopus-export-%s.json", time.Now().Format("20060102150405"))
	filePath := fmt.Sprintf("%s%c%s-%s", tmpDir, os.PathSeparator, taskID, fileName)

	ctx, cancel := context.WithCancel(context.Background())

	task := model.ExportTask{
		ID:        taskID,
		Status:    model.ExportStatusRunning,
		FilePath:  filePath,
		FileName:  fileName,
		CreatedAt: time.Now(),
	}

	doneCh := make(chan struct{})

	exportTasksMu.Lock()
	exportTasks[taskID] = &exportTaskState{task: task, cancel: cancel, doneCh: doneCh}
	exportTasksMu.Unlock()

	go runExport(ctx, taskID, filePath, doneCh, includeLogs, includeStats)

	return taskID, nil
}

// ExportCancel 取消导出任务
func ExportCancel(taskID string) error {
	exportTasksMu.RLock()
	state, ok := exportTasks[taskID]
	exportTasksMu.RUnlock()
	if !ok {
		return fmt.Errorf("task not found")
	}

	state.cancel()
	return nil
}

// ExportGetTask 获取任务信息
func ExportGetTask(taskID string) *model.ExportTask {
	exportTasksMu.RLock()
	state, ok := exportTasks[taskID]
	exportTasksMu.RUnlock()
	if !ok {
		return nil
	}
	task := state.task
	return &task
}

// ExportSubscribe 订阅任务进度
func ExportSubscribe(taskID string) chan model.ExportProgress {
	ch := make(chan model.ExportProgress, 256)

	exportSubscribersMu.Lock()
	exportSubscribers[taskID] = append(exportSubscribers[taskID], ch)
	exportSubscribersMu.Unlock()

	return ch
}

// ExportUnsubscribe 取消订阅
func ExportUnsubscribe(taskID string, ch chan model.ExportProgress) {
	exportSubscribersMu.Lock()
	subs := exportSubscribers[taskID]
	for i, s := range subs {
		if s == ch {
			exportSubscribers[taskID] = append(subs[:i], subs[i+1:]...)
			break
		}
	}
	if len(exportSubscribers[taskID]) == 0 {
		delete(exportSubscribers, taskID)
	}
	exportSubscribersMu.Unlock()
}

func notifyExportSubscribers(taskID string, progress model.ExportProgress) {
	exportSubscribersMu.RLock()
	subs := exportSubscribers[taskID]
	exportSubscribersMu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- progress:
		default:
		}
	}
}

// ExportGetDoneCh 获取任务完成信号 channel
func ExportGetDoneCh(taskID string) chan struct{} {
	exportTasksMu.RLock()
	state, ok := exportTasks[taskID]
	exportTasksMu.RUnlock()
	if !ok {
		return nil
	}
	return state.doneCh
}

// ExportGetFinalEvent 获取任务最终状态事件
func ExportGetFinalEvent(taskID string) model.ExportProgress {
	exportTasksMu.RLock()
	state, ok := exportTasks[taskID]
	exportTasksMu.RUnlock()
	if !ok {
		return model.ExportProgress{Status: model.ExportStatusError, Error: "task not found"}
	}
	return state.finalEvent
}

func runExport(ctx context.Context, taskID, filePath string, doneCh chan struct{}, includeLogs, includeStats bool) {
	// 给 SSE 连接留出时间
	select {
	case <-time.After(500 * time.Millisecond):
	case <-ctx.Done():
		close(doneCh)
		return
	}

	var err error
	defer func() {
		var finalProgress model.ExportProgress

		exportTasksMu.Lock()
		state, ok := exportTasks[taskID]
		if ok {
			if r := recover(); r != nil {
				state.task.Status = model.ExportStatusError
				state.task.Error = fmt.Sprintf("panic: %v", r)
				finalProgress = model.ExportProgress{Status: model.ExportStatusError, Error: state.task.Error}
			} else if ctx.Err() != nil {
				state.task.Status = model.ExportStatusCancelled
				os.Remove(filePath)
				finalProgress = model.ExportProgress{Status: model.ExportStatusCancelled}
			} else if err != nil {
				state.task.Status = model.ExportStatusError
				state.task.Error = err.Error()
				os.Remove(filePath)
				finalProgress = model.ExportProgress{Status: model.ExportStatusError, Error: err.Error()}
			} else {
				state.task.Status = model.ExportStatusDone
				finalProgress = model.ExportProgress{Status: model.ExportStatusDone}
			}
			state.finalEvent = finalProgress
		}
		exportTasksMu.Unlock()

		// 通知活跃订阅者
		notifyExportSubscribers(taskID, finalProgress)

		// 关闭完成信号
		close(doneCh)
	}()

	err = DBExportToFile(ctx, taskID, filePath, includeLogs, includeStats)
}

func exportCleanupLoop() {
	ticker := time.NewTicker(exportCleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		exportCleanup()
	}
}

func exportCleanup() {
	now := time.Now()

	exportTasksMu.Lock()
	defer exportTasksMu.Unlock()

	for id, state := range exportTasks {
		if now.Sub(state.task.CreatedAt) > exportTaskTTL {
			if state.task.FilePath != "" {
				if err := os.Remove(state.task.FilePath); err != nil && !os.IsNotExist(err) {
					log.Warnf("export cleanup: remove temp file %s failed: %v", state.task.FilePath, err)
				}
			}
			delete(exportTasks, id)
		}
	}
}

func generateID() string {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
