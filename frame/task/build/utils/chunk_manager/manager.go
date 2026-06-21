package chunk_manager

import (
	"github.com/Yeah114/Fatalder/define"
	"github.com/Yeah114/Fatalder/frame/task/build/utils/building_world"
)

// ChunkManager 按固定边长的区块组批量读取建筑范围内的区块。
type ChunkManager struct {
	world             *building_world.BuildingWorld
	dimension         define.Dimension
	chunkPosGenerator ChunkPosGenerator
	progress          int
	max               int
	chunkGroupSide    int
}

// ChunkManagerConfig 描述 ChunkManager 的构造参数。
type ChunkManagerConfig struct {
	// World 提供按建筑范围偏移后的区块读取能力。
	World *building_world.BuildingWorld
	// Dimension 是需要读取的世界维度。
	Dimension define.Dimension
	// ChunkPosGeneratorFunc 创建区块组遍历策略，留空时使用蛇形顺序。
	ChunkPosGeneratorFunc ChunkPosGeneratorFunc
	// Progress 是已处理的区块组数量，用于从断点继续。
	Progress int
	// ChunkGroupSide 是每组区块的边长，小于等于 0 时按 1 处理。
	ChunkGroupSide int
}

// New 创建一个按区块组分批读取结构数据的管理器。
func (c ChunkManagerConfig) New() *ChunkManager {
	world := c.World
	dimension := c.Dimension
	progress := c.Progress
	chunkGroupSide := c.ChunkGroupSide
	if chunkGroupSide <= 0 {
		chunkGroupSide = 1
	}

	chunkPosGeneratorFunc := c.ChunkPosGeneratorFunc
	if chunkPosGeneratorFunc == nil {
		chunkPosGeneratorFunc = NewSnakeChunkPosGenerator
	}

	size := world.Size()
	chunkXGroups := (size.ChunkXCount() + chunkGroupSide - 1) / chunkGroupSide
	chunkZGroups := (size.ChunkZCount() + chunkGroupSide - 1) / chunkGroupSide
	totalGroups := chunkXGroups * chunkZGroups

	manager := &ChunkManager{
		world:          world,
		dimension:      dimension,
		progress:       progress,
		max:            totalGroups,
		chunkGroupSide: chunkGroupSide,
	}
	manager.chunkPosGenerator = chunkPosGeneratorFunc(manager)
	return manager
}

// World 返回当前绑定的世界读取器。
func (c *ChunkManager) World() *building_world.BuildingWorld {
	return c.world
}

// Dimension 返回当前读取维度。
func (c *ChunkManager) Dimension() define.Dimension {
	return c.dimension
}

// ChunkGroupSide 返回每个区块组的边长。
func (c *ChunkManager) ChunkGroupSide() int {
	return c.chunkGroupSide
}

// ChunkGroupPos 返回指定进度索引对应的区块组坐标。
func (c *ChunkManager) ChunkGroupPos(index int) define.ChunkPos {
	return c.chunkPosGenerator.Index(index)
}
