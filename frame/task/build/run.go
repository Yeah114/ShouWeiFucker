package build

import (
	"context"
	"fmt"
)

// run 执行构建任务主流程。
//
// 当前阶段只实现普通方块构建；后续清理、NBT 方块、命令方块升级、等待区块加载等流程都应接入这里。
func (b *BuildTask) run(ctx context.Context) error {
	// 无论正常结束、暂停还是错误退出，都要释放当前任务上下文引用；
	// 如果 Start/Resume 已经创建了更新的上下文，finishTaskContext 会自动忽略旧上下文。
	defer b.finishTaskContext(ctx)

	// 区块组总数在任务初始化后不会变化，因此只在进入主流程时读取一次。
	progress, total := b.chunkManager.Progress()
	b.publish(EventNameRunStart, b.world.Size(), total)
	for ; progress < total; progress++ {
		// 在进入新组前先检查暂停/关闭请求，避免已经取消后继续读取世界或发送命令。
		if err := b.checkTaskContext(ctx); err != nil {
			if b.taskCanceled(ctx, err) {
				return nil
			}
			return fmt.Errorf("BuildTask.run: %w", err)
		}

		// 先按当前进度算出区块组坐标并移动机器人，保证后续读取和构建尽量发生在目标区块加载范围内。
		groupPos := b.chunkManager.ChunkGroupPos(progress)
		targetPos, err := b.moveBotToChunk(ctx, groupPos)
		if err != nil {
			if b.taskCanceled(ctx, err) {
				return nil
			}
			return fmt.Errorf("BuildTask.run: move bot to chunk: %w", err)
		}
		b.publish(EventNameRunChunkGroupMove, progress, groupPos, targetPos)

		if b.shouldWaitChunkLoad() {
			b.publish(EventNameRunChunkGroupWaitLoadStart, progress, groupPos)
			if err := b.waitChunkLoad(ctx, progress, groupPos); err != nil {
				if b.taskCanceled(ctx, err) {
					return nil
				}
				return fmt.Errorf("BuildTask.run: wait chunk load: %w", err)
			}
			b.publish(EventNameRunChunkGroupWaitLoadFinish, progress, groupPos)
		}

		b.publish(EventNameRunChunkGroupStart, progress)

		// ChunkManager.NextChunkGroup 会推进内部区块组游标，并返回当前组坐标、方块数据和 NBT 数据。
		// 真正可持久化的断点仍然只依赖 CurrentChunk；暂停后 Resume 会重新 Init 并按 CurrentChunk 重建游标。
		_, chunks, nbts, err := b.chunkManager.NextChunkGroup()
		if err != nil {
			return fmt.Errorf("BuildTask.run: next chunk group: %w", err)
		}
		b.publish(EventNameRunChunkGroupLoaded, progress, chunks, nbts)

		// 当前阶段只生成普通方块命令；后续 NBT 方块和命令方块流程应在这里之后接入。
		// 这里不会修改 checkpoint，命令全部发送完成后才认为这一组真正完成。
		commands := b.blockBuilder.BuildCommands(chunks)
		b.publish(EventNameRunCommandsGenerated, progress, len(commands))

		// 命令发送统一走封装方法，保证限速器对所有构建命令生效。
		for _, command := range commands {
			// 每条命令前都检查一次取消，暂停时最多只会多完成当前正在发送的一条命令。
			if err := b.checkTaskContext(ctx); err != nil {
				if b.taskCanceled(ctx, err) {
					return nil
				}
				return fmt.Errorf("BuildTask.run: %w", err)
			}
			if err := b.sendSettingsCommand(ctx, command, false); err != nil {
				if b.taskCanceled(ctx, err) {
					return nil
				}
				return fmt.Errorf("BuildTask.run: send build command: %w", err)
			}
			b.publish(EventNameRunCommandSent, progress, command)
		}
		// 只有当前区块组的全部命令发送成功后，才推进持久化断点。
		// 如果中途暂停或失败，Resume 会从这个区块组重新开始，避免跳过未完成内容。
		b.updateCurrentChunk(progress + 1)
		b.publish(EventNameRunChunkGroupFinish, progress, len(commands))
	}
	b.publish(EventNameRunFinish, progress)
	return nil
}

// updateCurrentChunk 将当前区块组进度换算回区块断点，便于任务恢复。
func (b *BuildTask) updateCurrentChunk(progress int) {
	b.CurrentChunk = progress * b.chunkGroupSide() * b.chunkGroupSide()
}
