package building_world

import (
	"fmt"

	"github.com/EmptyDea-Team/bedrock-world-operator/chunk"
	"github.com/EmptyDea-Team/bedrock-world-operator/world"
	"github.com/Yeah114/Fatalder/define"
)

const (
	denyBlockName   = "minecraft:deny"
	borderBlockName = "minecraft:border_block"
)

type BuildingWorldConfig struct {
	BedrockWorld         *world.BedrockWorld
	StartPos             define.BlockPos
	EndPos               define.BlockPos
	AutoPlaceDenyBlock   bool
	AutoPlaceBorderBlock bool
}

type BuildingWorld struct {
	BuildingWorldConfig

	airRuntimeID    uint32
	denyRuntimeID   uint32
	borderRuntimeID uint32
}

func NewBuildingWorld(config BuildingWorldConfig) (*BuildingWorld, error) {
	if config.BedrockWorld == nil {
		return nil, fmt.Errorf("bedrock world is nil")
	}
	config.StartPos, config.EndPos = config.StartPos.SortStartAndEndPos(config.EndPos)
	table := config.BedrockWorld.BlockRuntimeIDTable()
	if table == nil {
		return nil, fmt.Errorf("block runtime id table is nil")
	}

	w := &BuildingWorld{
		BuildingWorldConfig: config,
		airRuntimeID:        table.AirRuntimeID(),
	}
	if config.AutoPlaceDenyBlock {
		runtimeID, found := table.StateToRuntimeID(denyBlockName, map[string]any{})
		if !found {
			return nil, fmt.Errorf("block state not found: %s", denyBlockName)
		}
		w.denyRuntimeID = runtimeID
	}
	if config.AutoPlaceBorderBlock {
		runtimeID, found := table.StateToRuntimeID(borderBlockName, map[string]any{})
		if !found {
			return nil, fmt.Errorf("block state not found: %s", borderBlockName)
		}
		w.borderRuntimeID = runtimeID
	}
	return w, nil
}

// World 返回底层基岩版世界。
func (w *BuildingWorld) World() *world.BedrockWorld {
	return w.BedrockWorld
}

func (w *BuildingWorld) LoadChunk(dm define.Dimension, position define.ChunkPos) (c *chunk.Chunk, exists bool, err error) {
	if w.canLoadChunkDirectly() {
		return w.BedrockWorld.LoadChunk(dm, w.sourceChunkPos(position))
	}
	sourceChunks, err := w.loadSourceChunks(dm, w.sourceChunkPositions(position))
	if err != nil {
		return nil, true, err
	}

	c = chunk.NewChunk(w.airRuntimeID, dm.Range())
	for x := range 16 {
		for z := range 16 {
			for y := dm.Range()[0]; y <= dm.Range()[1]; y++ {
				target := targetBlockPos(position, x, y, z)
				if runtimeID, ok := w.generatedBlockAt(target); ok {
					exists = true
					c.SetBlock(uint8(x), int16(y), uint8(z), 0, runtimeID)
					continue
				}

				source := w.sourceBlockPos(target)
				if w.outOfSourceBounds(source) || source.Y() < dm.Range()[0] || source.Y() > dm.Range()[1] {
					continue
				}
				sourceChunk := sourceChunks[blockPosToChunkPos(source)]
				if sourceChunk == nil {
					continue
				}
				exists = true
				c.SetBlock(
					uint8(x),
					int16(y),
					uint8(z),
					0,
					sourceChunk.Block(uint8(floorMod(source.X(), 16)), int16(source.Y()), uint8(floorMod(source.Z(), 16)), 0),
				)
			}
		}
	}
	return c, exists, nil
}

func (w *BuildingWorld) Size() define.Size {
	size := w.StartPos.BlockSize(w.EndPos)
	size.Width++
	size.Height++
	size.Length++
	if w.AutoPlaceDenyBlock {
		size.Height++
	}
	if w.AutoPlaceBorderBlock {
		size.Width += 2
		size.Length += 2
	}
	return size
}

func (w *BuildingWorld) LoadSubChunk(dm define.Dimension, position define.SubChunkPos) *chunk.SubChunk {
	if w.canLoadChunkDirectly() {
		return w.BedrockWorld.LoadSubChunk(dm, w.sourceSubChunkPos(position))
	}
	sourceChunks, err := w.loadSourceChunks(dm, w.sourceChunkPositions(define.ChunkPos{position[0], position[2]}))
	if err != nil {
		return nil
	}

	sub := chunk.NewSubChunk(w.airRuntimeID)
	baseY := int(position[1] << 4)
	for x := range 16 {
		for y := range 16 {
			for z := range 16 {
				blockY := baseY + y
				if blockY < dm.Range()[0] || blockY > dm.Range()[1] {
					continue
				}
				target := define.BlockPos{
					int(position[0]<<4) + x,
					blockY,
					int(position[2]<<4) + z,
				}
				if runtimeID, ok := w.generatedBlockAt(target); ok {
					sub.SetBlock(byte(x), byte(y), byte(z), 0, runtimeID)
					continue
				}

				source := w.sourceBlockPos(target)
				if w.outOfSourceBounds(source) || source.Y() < dm.Range()[0] || source.Y() > dm.Range()[1] {
					continue
				}
				sourceChunk := sourceChunks[blockPosToChunkPos(source)]
				if sourceChunk == nil {
					continue
				}
				sub.SetBlock(
					byte(x),
					byte(y),
					byte(z),
					0,
					sourceChunk.Block(uint8(floorMod(source.X(), 16)), int16(source.Y()), uint8(floorMod(source.Z(), 16)), 0),
				)
			}
		}
	}
	if sub.Empty() {
		return nil
	}
	return sub
}

func (w *BuildingWorld) LoadNBT(dm define.Dimension, position define.ChunkPos) ([]map[string]any, error) {
	sourcePositions := w.sourceChunkPositions(position)
	result := make([]map[string]any, 0)
	for _, sourcePosition := range sourcePositions {
		nbts, err := w.BedrockWorld.LoadNBT(dm, sourcePosition)
		if err != nil {
			return nil, err
		}
		for _, data := range nbts {
			shifted, ok := w.shiftNBTIntoTargetChunk(data, position)
			if ok {
				result = append(result, shifted)
			}
		}
	}
	return result, nil
}

func (w *BuildingWorld) generatedBlockAt(target define.BlockPos) (uint32, bool) {
	if w.AutoPlaceDenyBlock && target.Y() == w.StartPos.Y() && !w.isBorderColumn(target) {
		return w.denyRuntimeID, true
	}
	if w.AutoPlaceBorderBlock && target.Y() == w.StartPos.Y() && w.isBorderColumn(target) {
		return w.borderRuntimeID, true
	}
	return 0, false
}

func (w *BuildingWorld) sourceBlockPos(target define.BlockPos) define.BlockPos {
	source := target
	source[0] += w.StartPos.X()
	source[2] += w.StartPos.Z()
	if w.AutoPlaceDenyBlock {
		source[1]--
	}
	if w.AutoPlaceBorderBlock {
		source[0]--
		source[2]--
	}
	return source
}

func (w *BuildingWorld) outOfSourceBounds(pos define.BlockPos) bool {
	return pos.X() < w.StartPos.X() ||
		pos.X() > w.EndPos.X() ||
		pos.Y() < w.StartPos.Y() ||
		pos.Y() > w.EndPos.Y() ||
		pos.Z() < w.StartPos.Z() ||
		pos.Z() > w.EndPos.Z()
}

func (w *BuildingWorld) sourceChunkPos(target define.ChunkPos) define.ChunkPos {
	source := w.sourceBlockPos(define.BlockPos{int(target[0] << 4), 0, int(target[1] << 4)})
	return blockPosToChunkPos(source)
}

func (w *BuildingWorld) sourceSubChunkPos(target define.SubChunkPos) define.SubChunkPos {
	source := w.sourceBlockPos(define.BlockPos{int(target[0] << 4), int(target[1] << 4), int(target[2] << 4)})
	return define.SubChunkPos{
		int32(floorDiv(source.X(), 16)),
		int32(floorDiv(source.Y(), 16)),
		int32(floorDiv(source.Z(), 16)),
	}
}

func (w *BuildingWorld) sourceChunkPositions(target define.ChunkPos) []define.ChunkPos {
	positions := make(map[define.ChunkPos]struct{}, 4)
	for _, x := range []int{0, 15} {
		for _, z := range []int{0, 15} {
			source := w.sourceBlockPos(targetBlockPos(target, x, 0, z))
			positions[blockPosToChunkPos(source)] = struct{}{}
		}
	}
	result := make([]define.ChunkPos, 0, len(positions))
	for position := range positions {
		result = append(result, position)
	}
	return result
}

func (w *BuildingWorld) loadSourceChunks(dm define.Dimension, positions []define.ChunkPos) (map[define.ChunkPos]*chunk.Chunk, error) {
	chunks := make(map[define.ChunkPos]*chunk.Chunk, len(positions))
	for _, position := range positions {
		c, exists, err := w.BedrockWorld.LoadChunk(dm, position)
		if err != nil {
			return nil, err
		}
		if !exists || c == nil {
			continue
		}
		chunks[position] = c
	}
	return chunks, nil
}

func (w *BuildingWorld) shiftNBTIntoTargetChunk(data map[string]any, target define.ChunkPos) (map[string]any, bool) {
	x, y, z, ok := define.GetPosFromNBT(data)
	if !ok {
		return nil, false
	}
	targetPos := define.BlockPos{x, y, z}
	targetPos[0] -= w.StartPos.X()
	targetPos[2] -= w.StartPos.Z()
	if w.AutoPlaceDenyBlock {
		targetPos[1]++
	}
	if w.AutoPlaceBorderBlock {
		targetPos[0]++
		targetPos[2]++
	}

	if blockPosToChunkPos(targetPos) != target {
		return nil, false
	}
	shifted := make(map[string]any, len(data))
	for key, value := range data {
		shifted[key] = value
	}
	shifted["x"] = int32(targetPos.X())
	shifted["y"] = int32(targetPos.Y())
	shifted["z"] = int32(targetPos.Z())
	return shifted, true
}

func (w *BuildingWorld) canLoadChunkDirectly() bool {
	return !w.AutoPlaceDenyBlock &&
		!w.AutoPlaceBorderBlock &&
		floorMod(w.StartPos.X(), 16) == 0 &&
		floorMod(w.StartPos.Z(), 16) == 0 &&
		w.StartPos.Y() <= define.WorldRange[0] &&
		w.EndPos.Y() >= define.WorldRange[1]
}

func (w *BuildingWorld) isBorderColumn(pos define.BlockPos) bool {
	maxX := w.EndPos.X() - w.StartPos.X() + 2
	maxZ := w.EndPos.Z() - w.StartPos.Z() + 2
	return pos.X() == 0 || pos.Z() == 0 || pos.X() == maxX || pos.Z() == maxZ
}

func targetBlockPos(chunkPos define.ChunkPos, x, y, z int) define.BlockPos {
	return define.BlockPos{
		int(chunkPos[0]<<4) + x,
		y,
		int(chunkPos[1]<<4) + z,
	}
}

func blockPosToChunkPos(pos define.BlockPos) define.ChunkPos {
	return define.ChunkPos{
		int32(floorDiv(pos.X(), 16)),
		int32(floorDiv(pos.Z(), 16)),
	}
}

func floorDiv(value, divisor int) int {
	result := value / divisor
	if value%divisor != 0 && (value < 0) != (divisor < 0) {
		result--
	}
	return result
}

func floorMod(value, divisor int) int {
	result := value % divisor
	if result < 0 {
		result += divisor
	}
	return result
}
