package build

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/EmptyDea-Team/bedrock-world-operator/block"
	bwo_world "github.com/EmptyDea-Team/bedrock-world-operator/world"
	"github.com/Yeah114/Fatalder/define"
	build_utils "github.com/Yeah114/Fatalder/frame/task/build/utils"
	"github.com/Yeah114/Fatalder/frame/task/build/utils/block_builder"
	"github.com/Yeah114/Fatalder/frame/task/build/utils/building_world"
	"github.com/Yeah114/Fatalder/frame/task/build/utils/chunk_manager"
	"go.uber.org/ratelimit"
)

const (
	defaultSpeed          = 3000
	defaultChunkGroupSide = 2
)

// init 初始化构建任务运行时依赖。
//
// 该方法只负责准备构建阶段需要复用的内部对象，不会发送任何游戏指令，也不会推进任务进度。
// 调用方应当在真正开始构建前显式调用它；NewTask 不会自动触发初始化。
//
// 初始化内容包括：
//   - 创建命令限速器。
//   - 打开源基岩版世界目录或压缩包。
//   - 将源世界包装成支持起止坐标裁剪、deny 和 border 生成的 BuildingWorld。
//   - 按配置和断点创建区块组管理器。
func (b *BuildTask) init() error {
	b.publish(EventNameInitStart)
	b.limiter = ratelimit.New(b.speed())

	b.publish(EventNameInitOpenWorld, b.WorldPath)
	bedrockWorld, err := b.openBedrockWorld()
	if err != nil {
		return fmt.Errorf("BuildTask.init: %w", err)
	}

	w, err := building_world.NewBuildingWorld(building_world.BuildingWorldConfig{
		BedrockWorld:         bedrockWorld,
		StartPos:             b.WorldStartPos,
		EndPos:               b.WorldEndPos,
		AutoPlaceDenyBlock:   b.EnableAutoPlaceDenyBlock,
		AutoPlaceBorderBlock: b.EnableAutoPlaceBorderBlock,
	})
	if err != nil {
		_ = bedrockWorld.CloseWorld()
		return fmt.Errorf("BuildTask.init: init build world: %w", err)
	}

	b.world = w
	b.chunkManager = chunk_manager.ChunkManagerConfig{
		World:          w,
		Dimension:      b.WorldDimension,
		Progress:       b.startGroupIndex(w.Size()),
		ChunkGroupSide: b.chunkGroupSide(),
	}.New()
	b.blockBuilder = block_builder.BlockBuilderConfig{
		ChunkManager:         b.chunkManager,
		RuntimeIDTable:       w.World().BlockRuntimeIDTable(),
		DisableFillBuildMode: b.DisableAutoFillBuildMode,
		StartPos:             b.StartPos,
	}.New()
	b.publish(EventNameInitFinish)
	return nil
}

// Init 初始化构建任务运行时依赖，并保证同一个任务实例只会执行一次实际初始化。
func (b *BuildTask) Init() error {
	var err error
	b.initOnce.Do(func() {
		err = b.init()
	})
	if err != nil {
		return fmt.Errorf("BuildTask.Init: %w", err)
	}
	return nil
}

// openBedrockWorld 根据 WorldPath 类型打开源世界。
//
// WorldPath 指向普通文件时会按只读压缩包处理，并使用 world.OpenZip 打开；
// WorldPath 指向目录时会按普通基岩版存档目录处理，并使用 world.Open 打开。
// 这里不依赖文件扩展名，因此 .mcworld、.zip 或其他文件名都只由文件类型决定。
func (b *BuildTask) openBedrockWorld() (*bwo_world.BedrockWorld, error) {
	table := block.NewBlockRuntimeIDTable(true)

	info, err := os.Stat(b.WorldPath)
	if err != nil {
		return nil, fmt.Errorf("BuildTask.openBedrockWorld: stat world path %q: %w", b.WorldPath, err)
	}
	if info.Mode().IsRegular() {
		db, err := bwo_world.OpenZip(b.WorldPath, nil, table)
		if err != nil {
			return nil, fmt.Errorf("BuildTask.openBedrockWorld: open zipped world %q: %w", b.WorldPath, err)
		}
		return db, nil
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("BuildTask.openBedrockWorld: world path %q is not a regular file or directory", b.WorldPath)
	}

	db, err := bwo_world.Open(b.WorldPath, nil, table)
	if err != nil {
		return nil, fmt.Errorf("BuildTask.openBedrockWorld: open world %q: %w", b.WorldPath, err)
	}
	return db, nil
}

// speed 返回命令发送速度，配置为空或非法时使用默认值。
func (b *BuildTask) speed() int {
	if b.Speed == nil || *b.Speed <= 0 {
		return defaultSpeed
	}
	return *b.Speed
}

// chunkGroupSide 返回区块组边长，配置为空或非法时使用默认值。
func (b *BuildTask) chunkGroupSide() int {
	if b.ChunkGroupSide == nil || *b.ChunkGroupSide <= 0 {
		return defaultChunkGroupSide
	}
	return *b.ChunkGroupSide
}

// startGroupIndex 将配置中的区块进度转换为区块组进度。
func (b *BuildTask) startGroupIndex(size define.Size) int {
	chunkIndex := b.startChunkIndex(size.ChunkCount())
	chunkGroupSize := b.chunkGroupSide() * b.chunkGroupSide()
	if chunkGroupSize <= 0 {
		return 0
	}
	return chunkIndex / chunkGroupSize
}

// startChunkIndex 解析任务起始区块序号。
//
// CurrentChunk 优先级高于 Progress。Progress 支持纯数字和百分比两种形式：
// 纯数字表示区块序号，百分比表示从总区块数换算出的区块序号。
func (b *BuildTask) startChunkIndex(chunkCount int) int {
	if chunkCount <= 0 {
		return 0
	}
	if b.CurrentChunk > 0 {
		return build_utils.ClampIndex(b.CurrentChunk, chunkCount)
	}

	progress := strings.TrimSpace(b.Progress)
	if progress == "" {
		return 0
	}
	if strings.HasSuffix(progress, "%") {
		value := strings.TrimSpace(strings.TrimSuffix(progress, "%"))
		percent, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return 0
		}
		return build_utils.ClampIndex(int(float64(chunkCount)*percent/100), chunkCount)
	}

	index, err := strconv.Atoi(progress)
	if err != nil {
		return 0
	}
	return build_utils.ClampIndex(index, chunkCount)
}
