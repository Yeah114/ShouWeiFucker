package build

import (
	"context"
	"fmt"
)

// Start 初始化任务并从当前断点开始执行构建任务。
func (b *BuildTask) Start() error {
	if err := b.Init(); err != nil {
		return fmt.Errorf("BuildTask.Start: %w", err)
	}
	if err := b.run(context.Background()); err != nil {
		return fmt.Errorf("BuildTask.Start: %w", err)
	}
	return nil
}

// run 执行构建任务主流程。
//
// 当前阶段只实现普通方块构建；后续清理、NBT 方块、命令方块升级、等待区块加载等流程都应接入这里。
func (b *BuildTask) run(ctx context.Context) error {
	for {
		progress, total := b.chunkManager.Progress()
		if progress >= total {
			return nil
		}

		chunks, _, err := b.chunkManager.NextChunkGroup()
		if err != nil {
			return fmt.Errorf("BuildTask.run: next chunk group: %w", err)
		}
		b.updateCurrentChunk()
		commands := b.blockBuilder.BuildCommands(chunks)

		for _, command := range commands {
			if err := b.sendSettingsCommand(ctx, command, false); err != nil {
				return fmt.Errorf("BuildTask.run: send build command: %w", err)
			}
		}
	}
}

// updateCurrentChunk 将当前区块组进度换算回区块断点，便于任务恢复。
func (b *BuildTask) updateCurrentChunk() {
	progress, _ := b.chunkManager.Progress()
	b.CurrentChunk = progress * b.chunkGroupSide() * b.chunkGroupSide()
}
