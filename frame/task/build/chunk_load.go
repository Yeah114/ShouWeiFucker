package build

import (
	"context"
	"errors"
	"fmt"
	"time"

	packet_pb "github.com/EmptyDea-Team/EmptyDea-core-api/pb/minecraft/protocol/packet"
	build_utils "github.com/Yeah114/Fatalder/frame/task/build/utils"

	"github.com/Yeah114/Fatalder/define"
)

const (
	chunkLoadProbeTimeout = 500 * time.Millisecond
	chunkLoadProbeDelay   = 100 * time.Millisecond
	chunkLoadOutOfWorld   = "commands.fill.outOfWorld"
)

// shouldWaitChunkLoad 返回当前任务是否需要在构建前等待目标区块加载完成。
func (b *BuildTask) shouldWaitChunkLoad() bool {
	return !b.DisableAutoWaitChunkLoad
}

// chunkLoadBounds 计算当前区块组等待加载时使用的目标世界坐标范围。
//
// 这里接收的是区块组坐标，不是源世界中的普通区块坐标。范围会覆盖整个区块组，
// 供 fill keep 探测使用。
func (b *BuildTask) chunkLoadBounds(groupPos define.ChunkPos) (startX, y, startZ, endX, endZ int) {
	groupWidth := b.chunkGroupSide() * 16
	startX = b.StartPos.X() + int(groupPos.X())*groupWidth
	startZ = b.StartPos.Z() + int(groupPos.Z())*groupWidth
	y = b.StartPos.Y() - build_utils.Mod(b.StartPos.Y(), 16)
	endX = startX + groupWidth - 1
	endZ = startZ + groupWidth - 1
	return startX, y, startZ, endX, endZ
}

// waitChunkLoad 通过 fill keep 探测当前区块组是否已被服务器加载。
//
// 未加载区块通常会返回 commands.fill.outOfWorld；一旦返回其他结果，说明这片区域已经可以访问。
func (b *BuildTask) waitChunkLoad(ctx context.Context, groupPos define.ChunkPos) error {
	if !b.shouldWaitChunkLoad() {
		return nil
	}

	startX, y, startZ, endX, endZ := b.chunkLoadBounds(groupPos)
	command := fmt.Sprintf("fill %d %d %d %d %d %d air keep", startX, y, startZ, endX, y, endZ)

	b.publish(EventNameRunChunkGroupWaitLoadStart, groupPos)
	for attempt := 1; ; attempt++ {
		if err := b.checkTaskContext(ctx); err != nil {
			return fmt.Errorf("BuildTask.waitChunkLoad: %w", err)
		}

		resp, timeout, err := b.sendWSCommandWithTimeout(ctx, command, chunkLoadProbeTimeout)
		if err != nil && !timeout {
			return fmt.Errorf("BuildTask.waitChunkLoad: probe chunk load: %w", err)
		}

		ready := false
		message := ""
		if !timeout {
			messages := resp.GetOutputMessages()
			if len(messages) != 1 {
				ready = true
			} else {
				message = messages[0].GetMessage()
				ready = message != chunkLoadOutOfWorld
			}
		}
		b.publish(EventNameRunChunkGroupWaitLoadProbe, groupPos, attempt, ready, timeout, message)
		if ready {
			b.publish(EventNameRunChunkGroupWaitLoadFinish, groupPos)
			return nil
		}
		b.publish(EventNameRunChunkGroupWaitLoadRetry, groupPos, attempt, timeout, message)

		timer := time.NewTimer(chunkLoadProbeDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("BuildTask.waitChunkLoad: %w", ctx.Err())
		case <-timer.C:
		}
	}
}

// sendWSCommandWithTimeout 给当前 WebSocket 命令添加一次性超时。
func (b *BuildTask) sendWSCommandWithTimeout(ctx context.Context, command string, timeout time.Duration) (resp *packet_pb.CommandOutput, isTimeout bool, err error) {
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	output, err := b.sendWSCommandWithResp(probeCtx, command)
	if err == nil {
		return output, false, nil
	}
	if errors.Is(probeCtx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return nil, true, nil
	}
	return nil, false, fmt.Errorf("BuildTask.sendWSCommandWithTimeout: %w", err)
}
