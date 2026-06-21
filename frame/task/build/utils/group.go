package utils

import (
	"math"

	"github.com/Yeah114/Fatalder/define"
)

// GroupIndex 根据区块坐标和组边长计算其所在的区块组索引。
func GroupIndex(pos int32, groupSide int) int {
	return int(math.Floor(float64(pos) / float64(groupSide)))
}

// ChunkPosInGroup 根据组索引和组内偏移还原实际区块坐标。
func ChunkPosInGroup(groupIdx, offset, groupSide int) int32 {
	return int32(groupIdx*groupSide + offset)
}

// GroupPosByChunkPos 将单个区块坐标转换为所属区块组坐标。
func GroupPosByChunkPos(chunkPos define.ChunkPos, groupSide int) define.ChunkPos {
	groupCX := GroupIndex(chunkPos.X(), groupSide)
	groupCZ := GroupIndex(chunkPos.Z(), groupSide)
	return define.ChunkPos{int32(groupCX), int32(groupCZ)}
}
