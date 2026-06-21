package chunk_manager

import (
	"math"

	"github.com/Yeah114/Fatalder/define"
	build_utils "github.com/Yeah114/Fatalder/frame/task/build/utils"
)

// ChunkPosGenerator 根据进度索引生成需要处理的区块组坐标。
type ChunkPosGenerator interface {
	Index(n int) define.ChunkPos
}

// ChunkPosGeneratorFunc 根据 ChunkManager 创建区块组坐标生成器。
type ChunkPosGeneratorFunc func(cm *ChunkManager) ChunkPosGenerator

// SnakeChunkPosGenerator 按行蛇形顺序生成区块组坐标。
type SnakeChunkPosGenerator struct {
	cm *ChunkManager
}

// Index 按蛇形顺序返回第 n 个区块组坐标。
func (s *SnakeChunkPosGenerator) Index(n int) define.ChunkPos {
	size := s.cm.World().Size()
	groupSide := s.cm.ChunkGroupSide()
	xChunkNum := size.ChunkXCount()
	xGroupNum := int(math.Ceil(float64(xChunkNum) / float64(groupSide)))
	groupCZ := n / xGroupNum
	groupRowOffset := build_utils.Mod(n, xGroupNum)

	var groupCX int
	if groupCZ%2 == 0 {
		groupCX = groupRowOffset
	} else {
		groupCX = xGroupNum - 1 - groupRowOffset
	}

	return define.ChunkPos{int32(groupCX), int32(groupCZ)}
}

// NewSnakeChunkPosGenerator 创建默认的蛇形区块组坐标生成器。
func NewSnakeChunkPosGenerator(cm *ChunkManager) ChunkPosGenerator {
	return &SnakeChunkPosGenerator{cm: cm}
}
