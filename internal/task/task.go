package task

import (
	"context"
	"sync"
	"time"

	"github.com/charmbracelet/log"
)

type taskEntry struct {
	interval   time.Duration
	fn         func(context.Context)
	runOnStart bool
	updateCh   chan struct{}
}

var (
	tasks   = make(map[string]*taskEntry)
	tasksMu sync.RWMutex
)

// Register 保留禁用任务的定义，使间隔从 0 改回正数时可以恢复调度。
func Register(name string, interval time.Duration, runOnStart bool, fn func(context.Context)) {
	tasksMu.Lock()
	defer tasksMu.Unlock()
	if _, exists := tasks[name]; exists {
		return
	}
	tasks[name] = &taskEntry{
		interval:   interval,
		fn:         fn,
		runOnStart: runOnStart,
		updateCh:   make(chan struct{}, 1),
	}
}

func Update(name string, interval time.Duration) {
	tasksMu.Lock()
	defer tasksMu.Unlock()
	entry, exists := tasks[name]
	if !exists {
		log.Warnf("task %s not found", name)
		return
	}
	entry.interval = interval
	// 通知可以合并，但最新间隔始终保存在锁保护的状态里。
	select {
	case entry.updateCh <- struct{}{}:
	default:
	}
}

// Run 随上下文停止调度，并等待正在执行的任务退出。
func Run(ctx context.Context) {
	var workers sync.WaitGroup
	tasksMu.RLock()
	for _, entry := range tasks {
		workers.Go(func() { runTask(ctx, entry) })
	}
	tasksMu.RUnlock()
	workers.Wait()
}

func runTask(ctx context.Context, entry *taskEntry) {
	timer := time.NewTimer(time.Hour)
	defer timer.Stop()
	var ticks <-chan time.Time
	reset := func() {
		timer.Stop()
		ticks = nil
		tasksMu.RLock()
		interval := entry.interval
		tasksMu.RUnlock()
		if interval > 0 {
			timer.Reset(interval)
			ticks = timer.C
		}
	}
	reset()
	if ticks != nil && entry.runOnStart && ctx.Err() == nil {
		entry.fn(ctx)
		reset()
	}
	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
			return
		case <-entry.updateCh:
			reset()
		case <-ticks:
			// 同一任务不重叠执行，避免慢同步和统计落库累积并发调用。
			entry.fn(ctx)
			reset()
		}
	}
}
