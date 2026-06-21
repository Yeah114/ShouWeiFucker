package build

import (
	"context"
	"fmt"
)

// Start 初始化任务并从当前断点开始构建所有普通方块。
func (b *BuildTask) Start() error {
	if err := b.Init(); err != nil {
		return fmt.Errorf("BuildTask.Start: %w", err)
	}
	if err := b.buildAllBlocks(context.Background()); err != nil {
		return fmt.Errorf("BuildTask.Start: %w", err)
	}
	return nil
}

// buildAllBlocks 按区块组生成并发送构建命令。
//
// 当前实现只处理方块构建，不处理清理、NBT 方块、命令方块升级、等待区块加载等高级流程。
func (b *BuildTask) buildAllBlocks(ctx context.Context) error {
	for {
		progress, total := b.chunkManager.Progress()
		if progress >= total {
			return nil
		}

		commands, err := b.blockBuilder.NextChunkGroupCommands()
		if err != nil {
			return fmt.Errorf("BuildTask.buildAllBlocks: next chunk group commands: %w", err)
		}
		b.updateCurrentChunk()

		for _, command := range commands {
			if err := b.sendSettingsCommand(ctx, command, false); err != nil {
				return fmt.Errorf("BuildTask.buildAllBlocks: send build command: %w", err)
			}
		}
	}
}

// updateCurrentChunk 将当前区块组进度换算回区块断点，便于任务恢复。
func (b *BuildTask) updateCurrentChunk() {
	progress, _ := b.chunkManager.Progress()
	b.CurrentChunk = progress * b.chunkGroupSide() * b.chunkGroupSide()
}
