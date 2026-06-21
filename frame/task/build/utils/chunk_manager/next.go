package chunk_manager

import (
	"fmt"

	"github.com/EmptyDea-Team/bedrock-world-operator/chunk"
	"github.com/Yeah114/Fatalder/define"
	build_utils "github.com/Yeah114/Fatalder/frame/task/build/utils"
)

// ChunkGroup 读取指定索引对应的一组区块，不推进内部进度。
func (c *ChunkManager) ChunkGroup(index int) (define.ChunkPos, map[define.ChunkPos]*chunk.Chunk, error) {
	if c.world == nil || index < 0 || index >= c.max {
		return define.ChunkPos{}, nil, nil
	}

	groupPos := c.chunkPosGenerator.Index(index)
	allChunkPositions := make([]define.ChunkPos, 0)
	groupSide := c.chunkGroupSide
	size := c.World().Size()
	maxCX := int32(size.ChunkXCount())
	maxCZ := int32(size.ChunkZCount())

	groupCX, groupCZ := int(groupPos.X()), int(groupPos.Z())
	for zOffset := 0; zOffset < groupSide; zOffset++ {
		for xOffset := 0; xOffset < groupSide; xOffset++ {
			cx := build_utils.ChunkPosInGroup(groupCX, xOffset, groupSide)
			cz := build_utils.ChunkPosInGroup(groupCZ, zOffset, groupSide)
			if cx >= 0 && cx < maxCX && cz >= 0 && cz < maxCZ {
				allChunkPositions = append(allChunkPositions, define.ChunkPos{cx, cz})
			}
		}
	}

	if len(allChunkPositions) == 0 {
		return groupPos, nil, nil
	}

	chunks := make(map[define.ChunkPos]*chunk.Chunk, len(allChunkPositions))
	for _, chunkPos := range allChunkPositions {
		c, exists, err := c.world.LoadChunk(c.dimension, chunkPos)
		if err != nil {
			return define.ChunkPos{}, nil, fmt.Errorf("ChunkManager.ChunkGroup: load chunk %v: %w", chunkPos, err)
		}
		if exists {
			chunks[chunkPos] = c
		}
	}
	return groupPos, chunks, nil
}

// NextChunkGroup 获取下一组区块，并返回这组区块所属的组坐标。
func (c *ChunkManager) NextChunkGroup() (define.ChunkPos, map[define.ChunkPos]*chunk.Chunk, error) {
	groupPos, chunks, err := c.ChunkGroup(c.progress)
	if err != nil {
		return define.ChunkPos{}, nil, fmt.Errorf("ChunkManager.NextChunkGroup: %w", err)
	}
	if c.world != nil && c.progress < c.max {
		c.progress++
	}
	return groupPos, chunks, nil
}
