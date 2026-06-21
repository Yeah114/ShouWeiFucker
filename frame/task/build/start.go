package build

import (
	"context"
	"errors"
	"fmt"
)

// Start 初始化任务并从当前断点开始执行构建任务。
func (b *BuildTask) Start() error {
	if err := b.Init(); err != nil {
		return fmt.Errorf("BuildTask.Start: %w", err)
	}
	ctx := b.startRunContext()
	if err := b.run(ctx); err != nil {
		return fmt.Errorf("BuildTask.Start: %w", err)
	}
	return nil
}

// run 执行构建任务主流程。
//
// 当前阶段只实现普通方块构建；后续清理、NBT 方块、命令方块升级、等待区块加载等流程都应接入这里。
func (b *BuildTask) run(ctx context.Context) error {
	defer b.finishRunContext(ctx)

	// 区块组总数在任务初始化后不会变化，因此只在进入主流程时读取一次。
	progress, total := b.chunkManager.Progress()
	b.publish(EventNameRunStart, b.world.Size(), total)
	for ; progress < total; progress++ {
		if err := ctx.Err(); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return fmt.Errorf("BuildTask.run: %w", err)
		}

		b.publish(EventNameRunChunkGroupStart, progress)

		// 由 ChunkManager 统一推进区块组进度，并返回当前组的方块数据和 NBT 数据。
		groupPos, chunks, nbts, err := b.chunkManager.NextChunkGroup()
		if err != nil {
			return fmt.Errorf("BuildTask.run: next chunk group: %w", err)
		}
		targetPos, err := b.moveBotToChunk(ctx, groupPos)
		if err != nil {
			b.rollbackCurrentChunkGroup(progress)
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("BuildTask.run: move bot to chunk: %w", err)
		}
		b.publish(EventNameRunChunkGroupMove, progress, groupPos, targetPos)

		b.publish(EventNameRunChunkGroupLoaded, progress, chunks, nbts)

		// 当前阶段只生成普通方块命令；后续 NBT 方块和命令方块流程应在这里之后接入。
		commands := b.blockBuilder.BuildCommands(chunks)
		b.publish(EventNameRunCommandsGenerated, progress, len(commands))

		// 命令发送统一走封装方法，保证限速器对所有构建命令生效。
		for _, command := range commands {
			if err := ctx.Err(); err != nil {
				b.rollbackCurrentChunkGroup(progress)
				if errors.Is(err, context.Canceled) {
					return nil
				}
				return fmt.Errorf("BuildTask.run: %w", err)
			}
			if err := b.sendSettingsCommand(ctx, command, false); err != nil {
				b.rollbackCurrentChunkGroup(progress)
				if ctx.Err() != nil {
					return nil
				}
				return fmt.Errorf("BuildTask.run: send build command: %w", err)
			}
			b.publish(EventNameRunCommandSent, progress, command)
		}
		b.updateCurrentChunk(progress + 1)
		b.publish(EventNameRunChunkGroupFinish, progress, len(commands))
	}
	b.publish(EventNameRunFinish, progress)
	return nil
}

// startRunContext 创建本次运行的可取消上下文，供 Pause/Close 中断任务。
func (b *BuildTask) startRunContext() context.Context {
	b.runMu.Lock()
	defer b.runMu.Unlock()

	if b.runCancel != nil {
		b.runCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	b.runCtx = ctx
	b.runCancel = cancel
	return ctx
}

// finishRunContext 清理本次运行的取消函数。
func (b *BuildTask) finishRunContext(ctx context.Context) {
	b.runMu.Lock()
	defer b.runMu.Unlock()

	if b.runCtx != ctx {
		return
	}
	b.runCtx = nil
	b.runCancel = nil
}

// cancelRun 请求当前运行中的任务尽快停止。
func (b *BuildTask) cancelRun() {
	b.runMu.Lock()
	defer b.runMu.Unlock()

	if b.runCancel != nil {
		b.runCancel()
	}
}

// rollbackCurrentChunkGroup 将内部进度回退到当前区块组，避免恢复时跳过未完成的区块组。
func (b *BuildTask) rollbackCurrentChunkGroup(progress int) {
	b.chunkManager.SetProgress(progress)
}

// updateCurrentChunk 将当前区块组进度换算回区块断点，便于任务恢复。
func (b *BuildTask) updateCurrentChunk(progress int) {
	b.CurrentChunk = progress * b.chunkGroupSide() * b.chunkGroupSide()
}
