package build

import (
	"context"
	"errors"
	"fmt"
)

// startTaskContext 创建本次任务运行的可取消上下文，供 Pause/Close 中断任务。
func (b *BuildTask) startTaskContext() context.Context {
	b.taskMu.Lock()
	defer b.taskMu.Unlock()

	if b.taskCancel != nil {
		b.taskCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	b.taskCtx = ctx
	b.taskCancel = cancel
	return ctx
}

// finishTaskContext 清理本次任务运行的取消函数。
func (b *BuildTask) finishTaskContext(ctx context.Context) {
	b.taskMu.Lock()
	defer b.taskMu.Unlock()

	if b.taskCtx != ctx {
		return
	}
	b.taskCtx = nil
	b.taskCancel = nil
}

// checkTaskContext 检查任务上下文是否已取消。
func (b *BuildTask) checkTaskContext(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		if errors.Is(err, context.Canceled) {
			return context.Canceled
		}
		return fmt.Errorf("BuildTask.checkTaskContext: %w", err)
	}
	return nil
}

// cancelTask 请求当前运行中的任务尽快停止。
func (b *BuildTask) cancelTask() {
	b.taskMu.Lock()
	defer b.taskMu.Unlock()

	if b.taskCancel != nil {
		b.taskCancel()
	}
}

// Canceled 返回当前任务上下文是否已经被取消。
func (b *BuildTask) Canceled() bool {
	b.taskMu.Lock()
	defer b.taskMu.Unlock()

	return b.taskCtx != nil && b.taskCtx.Err() != nil
}
