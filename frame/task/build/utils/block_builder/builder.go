package block_builder

import (
	"fmt"

	"github.com/EmptyDea-Team/bedrock-world-operator/block"
	"github.com/EmptyDea-Team/bedrock-world-operator/chunk"
	"github.com/Yeah114/Fatalder/define"
	"github.com/Yeah114/Fatalder/frame/task/build/utils/chunk_fill"
	"github.com/Yeah114/Fatalder/utils"
)

type blockInfo struct {
	name  string
	state string
	isAir bool
}

// BlockBuilderConfig 描述区块组构建命令生成器的依赖。
type BlockBuilderConfig struct {
	// RuntimeIDTable 用于将方块运行时 ID 转换为命令中的方块名和状态。
	RuntimeIDTable *block.BlockRuntimeIDTable
	// DisableFillBuildMode 关闭 fill 贪心合并模式，改为逐方块 setblock。
	DisableFillBuildMode bool
	// StartPos 是目标世界中建筑局部坐标 (0,0,0) 对应的方块坐标。
	StartPos define.BlockPos
}

// BlockBuilder 按区块组生成可发送到游戏内的构建命令。
type BlockBuilder struct {
	BlockBuilderConfig

	blockInfoCache map[uint32]blockInfo
}

// New 创建区块组构建命令生成器。
func (c BlockBuilderConfig) New() *BlockBuilder {
	return &BlockBuilder{
		BlockBuilderConfig: c,
		blockInfoCache:     make(map[uint32]blockInfo),
	}
}

// BuildCommands 根据外部传入的区块生成构建命令。
func (b *BlockBuilder) BuildCommands(chunks map[define.ChunkPos]*chunk.Chunk) []string {
	if len(chunks) == 0 {
		return nil
	}

	startPos := b.chunkStartPos(minChunkPos(chunks))
	if !b.DisableFillBuildMode {
		return collectCommands(chunk_fill.GenerateChunksCommand(b.RuntimeIDTable, chunks, startPos))
	}
	return b.setBlockCommands(chunks, startPos)
}

func (b *BlockBuilder) chunkStartPos(chunkPos define.ChunkPos) define.BlockPos {
	return define.BlockPos{
		b.StartPos.X() + int(chunkPos.X())*16,
		b.StartPos.Y(),
		b.StartPos.Z() + int(chunkPos.Z())*16,
	}
}

func (b *BlockBuilder) setBlockCommands(chunks map[define.ChunkPos]*chunk.Chunk, startPos define.BlockPos) []string {
	commands := make([]string, 0)
	groupPos := minChunkPos(chunks)
	for chunkPos, c := range chunks {
		if c == nil {
			continue
		}
		offsetX := int(chunkPos.X()-groupPos.X()) * 16
		offsetZ := int(chunkPos.Z()-groupPos.Z()) * 16
		for x := range 16 {
			for z := range 16 {
				for y := c.Range().Min(); y <= c.Range().Max(); y++ {
					info := b.blockInfo(c.Block(uint8(x), int16(y), uint8(z), 0))
					if info.isAir {
						continue
					}
					commands = append(commands, fmt.Sprintf(
						"setblock %d %d %d %s %s\n",
						startPos.X()+offsetX+x,
						startPos.Y()+y,
						startPos.Z()+offsetZ+z,
						info.name,
						info.state,
					))
				}
			}
		}
	}
	return commands
}

func (b *BlockBuilder) blockInfo(runtimeID uint32) blockInfo {
	if info, ok := b.blockInfoCache[runtimeID]; ok {
		return info
	}

	airRuntimeID := b.RuntimeIDTable.AirRuntimeID()
	if runtimeID == airRuntimeID {
		info := blockInfo{name: "minecraft:air", state: "[]", isAir: true}
		b.blockInfoCache[runtimeID] = info
		return info
	}

	name, properties, found := b.RuntimeIDTable.RuntimeIDToState(runtimeID)
	state := utils.PropertiesToStateStr(properties)
	if !found {
		name = "minecraft:air"
		state = "[]"
	}

	info := blockInfo{name: name, state: state, isAir: runtimeID == airRuntimeID || name == "minecraft:air"}
	b.blockInfoCache[runtimeID] = info
	return info
}

func collectCommands(commandCh <-chan string) []string {
	commands := make([]string, 0)
	for command := range commandCh {
		commands = append(commands, command)
	}
	return commands
}

func minChunkPos(chunks map[define.ChunkPos]*chunk.Chunk) define.ChunkPos {
	var (
		result define.ChunkPos
		found  bool
	)
	for pos := range chunks {
		if !found || pos.X() < result.X() || (pos.X() == result.X() && pos.Z() < result.Z()) {
			result = pos
			found = true
		}
	}
	return result
}
