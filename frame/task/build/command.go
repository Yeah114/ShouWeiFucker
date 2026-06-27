package build

import (
	"context"
	"fmt"
	"math"
	"time"

	packet_pb "github.com/EmptyDea-Team/EmptyDea-core-api/pb/minecraft/protocol/packet"
	"github.com/Yeah114/Fatalder/define"
)

// takeCommandLimit 在发送命令前应用任务限速器。
func (b *BuildTask) takeCommandLimit() {
	return
	if b.limiter == nil {
		return
	}
	b.limiter.Take()
}

// sendSettingsCommand 发送设置类命令，并在发送前应用任务限速。
func (b *BuildTask) sendSettingsCommand(ctx context.Context, command string, dimensional bool) error {
	if !b.checkContext() {
		return fmt.Errorf("BuildTask.sendSettingsCommand: %w", context.Canceled)
	}
	b.takeCommandLimit()
	if err := b.frame.Client().GameInterface().Commands().SendSettingsCommand(ctx, command, dimensional); err != nil {
		return fmt.Errorf("BuildTask.sendSettingsCommand: %w", err)
	}
	return nil
}

// sendPlayerCommand 发送玩家命令，并在发送前应用任务限速。
func (b *BuildTask) sendPlayerCommand(ctx context.Context, command string) error {
	if !b.checkContext() {
		return fmt.Errorf("BuildTask.sendPlayerCommand: %w", context.Canceled)
	}
	b.takeCommandLimit()
	if err := b.frame.Client().GameInterface().Commands().SendPlayerCommand(ctx, command); err != nil {
		return fmt.Errorf("BuildTask.sendPlayerCommand: %w", err)
	}
	return nil
}

// sendWSCommand 发送 WebSocket 命令，并在发送前应用任务限速。
func (b *BuildTask) sendWSCommand(ctx context.Context, command string) error {
	if !b.checkContext() {
		return fmt.Errorf("BuildTask.sendWSCommand: %w", context.Canceled)
	}
	b.takeCommandLimit()
	if err := b.frame.Client().GameInterface().Commands().SendWSCommand(ctx, command); err != nil {
		return fmt.Errorf("BuildTask.sendWSCommand: %w", err)
	}
	return nil
}

// sendWSCommandWithResp 发送 WebSocket 命令并返回命令输出，在发送前应用任务限速。
func (b *BuildTask) sendWSCommandWithResp(ctx context.Context, command string) (*packet_pb.CommandOutput, error) {
	if !b.checkContext() {
		return nil, fmt.Errorf("BuildTask.sendWSCommandWithResp: %w", context.Canceled)
	}
	b.takeCommandLimit()
	resp, err := b.frame.Client().GameInterface().Commands().SendWSCommandWithResp(ctx, command)
	if err != nil {
		return nil, fmt.Errorf("BuildTask.sendWSCommandWithResp: %w", err)
	}
	return resp, nil
}

// sendWSCommandWithTimeout 给当前 WebSocket 命令添加一次性超时。
func (b *BuildTask) sendWSCommandWithTimeout(ctx context.Context, command string, timeout time.Duration) (resp *packet_pb.CommandOutput, isTimeout bool, err error) {
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	output, err := b.sendWSCommandWithResp(probeCtx, command)
	if err == nil {
		return output, false, nil
	}
	return nil, false, fmt.Errorf("BuildTask.sendWSCommandWithTimeout: %w", err)
}

// sendChat 发送聊天消息，并在发送前应用任务限速。
func (b *BuildTask) sendChat(ctx context.Context, content string) error {
	if !b.checkContext() {
		return fmt.Errorf("BuildTask.sendChat: %w", context.Canceled)
	}
	b.takeCommandLimit()
	if err := b.frame.Client().GameInterface().Commands().SendChat(ctx, content); err != nil {
		return fmt.Errorf("BuildTask.sendChat: %w", err)
	}
	return nil
}

// moveBotToChunkGroup 将机器人移动到目标区块组中心。
func (b *BuildTask) moveBotToChunkGroup(ctx context.Context, groupPos define.ChunkPos) (define.BlockPos, error) {
	pos := b.chunkGroupTargetPos(groupPos)
	if err := b.sendPlayerCommand(ctx, fmt.Sprintf("tp @s %d %d %d", pos.X(), pos.Y(), pos.Z())); err != nil {
		return define.BlockPos{}, fmt.Errorf("BuildTask.moveBotToChunkGroup: %w", err)
	}
	return pos, nil
}

// chunkGroupTargetPos 将区块组坐标转换成目标世界中的区块组中心坐标。
func (b *BuildTask) chunkGroupTargetPos(groupPos define.ChunkPos) define.BlockPos {
	groupWidth := float64(b.chunkGroupSide() * 16)
	return define.BlockPos{
		int(math.Floor(float64(b.StartPos.X()+int(groupPos.X())*b.chunkGroupSide()*16) + groupWidth/2 - 0.5)),
		define.WorldRange[1],
		int(math.Floor(float64(b.StartPos.Z()+int(groupPos.Z())*b.chunkGroupSide()*16) + groupWidth/2 - 0.5)),
	}
}
