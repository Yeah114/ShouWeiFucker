package build

import (
	"context"
	"sync"

	"github.com/Yeah114/Fatalder/define"
	"github.com/Yeah114/Fatalder/frame/task/build/utils/block_builder"
	"github.com/Yeah114/Fatalder/frame/task/build/utils/building_world"
	"github.com/Yeah114/Fatalder/frame/task/build/utils/chunk_manager"
	"go.uber.org/ratelimit"
)

// BuildTaskWorldConfig 源世界基础区域配置
type BuildTaskWorldConfig struct {
	// WorldPath 源世界资源路径，支持存档目录、.mcworld、.zip 压缩包
	WorldPath string `config:"world_path"`
	// WorldStartPos 源世界内待构建区域起始方块坐标
	WorldStartPos define.BlockPos `config:"world_start_pos"`
	// WorldEndPos 源世界内待构建区域结束方块坐标，与 WorldStartPos 划定构建区域范围
	WorldEndPos define.BlockPos `config:"world_end_pos"`
	// WorldDimension 源世界构建区域所属维度
	WorldDimension define.Dimension `config:"world_dimension"`
}

// BuildTaskAutoConfig 构建自动行为配置
type BuildTaskAutoConfig struct {
	// EnableAutoCleanBlock 开启自动清理方块
	EnableAutoCleanBlock bool `config:"enable_auto_clean_block"`
	// DisableAutoWaitChunkLoad 关闭自动等待区块加载
	DisableAutoWaitChunkLoad bool `config:"disable_auto_wait_chunk_load"`
	// DisableAutoFillBuildMode 关闭 fill 贪心构建模式
	DisableAutoFillBuildMode bool `config:"disable_auto_fill_build_mode"`
	// DisableAutoCleanItem 关闭自动清理掉落物
	DisableAutoCleanItem bool `config:"disable_auto_clean_item"`
	// DisableAutoCommandBlocksDisabled 关闭自动禁用命令方块运行
	DisableAutoCommandBlocksDisabled bool `config:"disable_auto_command_blocks_disabled"`
	// DisableAutoUpgradeCommandBlock 关闭自动升级旧命令
	DisableAutoUpgradeCommandBlock bool `config:"disable_auto_upgrade_command_block"`
	// EnableAutoPlaceDenyBlock 开启自动放置禁止方块
	EnableAutoPlaceDenyBlock bool `config:"enable_auto_place_deny_block"`
	// EnableAutoPlaceBorderBlock 开启自动放置边界方块
	EnableAutoPlaceBorderBlock bool `config:"enable_auto_place_border_block"`
	// DisableAutoEnterFixMode 关闭自动进入修补模式
	DisableAutoEnterFixMode bool `config:"disable_auto_enter_fix_mode"`
}

type BuildTaskBuildConfig struct {
	// Progress 纯数字代表起始区块进度，带 % 后缀代表起始百分比进度，默认0
	Progress string `config:"progress"`
	// ChunkGroupSide 区块组边长，默认值为 2
	ChunkGroupSide *int `config:"chunk_group_side"`
	// DisableGameProgress 关闭游戏内进度展示
	DisableGameProgress bool `config:"disable_game_progress"`
	// IgnoreCommandBlock 忽略命令方块数据
	IgnoreCommandBlock bool `config:"ignore_command_block"`
	// IgnoreOtherNBTBlock 忽略其他NBT方块数据
	IgnoreOtherNBTBlock bool `config:"ignore_other_nbt_block"`
}

type BuildTaskAdvConfig struct {
	// EnterFixModeDirectly 直接进入修补模式
	EnterFixModeDirectly bool `config:"enter_fix_mode_directly"`
	// GameProgressRefreshDelay 进度刷新延迟(秒)，默认值为 0.5
	GameProgressRefreshDelay *float64 `config:"game_progress_refresh_delay"`
	// ConsoleWorldPos 控制台的世界坐标，默认值为 -50, 0, -50
	ConsoleWorldPos *define.BlockPos `config:"console_world_pos"`
	// UseTickingArea 使用常加载区域辅助加载区块
	UseTickingArea bool `config:"use_ticking_area"`
	// PreWaitNextChunkLoad 预等待下一个区块加载
	PreWaitNextChunkLoad bool `config:"pre_wait_next_chunk_load"`
	// PreHandleNextChunkGroup 预处理下一个区块组
	PreHandleNextChunkGroup bool `config:"pre_handle_next_chunk_group"`
	// FixModeTimeout 修补模式超时(秒)，默认值为 10.0
	FixModeTimeout *float64 `config:"fix_mode_timeout"`
}

// BuildTaskConfig 完整构建任务总配置
type BuildTaskConfig struct {
	BuildTaskWorldConfig `config:",squash"`
	BuildTaskAutoConfig  `config:",squash"`
	BuildTaskBuildConfig `config:",squash"`
	BuildTaskAdvConfig   `config:",squash"`

	// StartPos 目标世界的构建起始坐标
	StartPos define.BlockPos `config:"start_pos"`
	// Dimension 目标构建维度
	Dimension define.Dimension `config:"dimension"`
	// Speed 命令发送速度，默认值为 3000
	Speed *int `config:"speed"`
}

// BuildTaskCheckpoint 构建任务断点存档信息
type BuildTaskCheckpoint struct {
	// CurrentChunk 当前处理到的区块序号
	CurrentChunk int `checkpoint:"current_chunk"`
}

// BuildTask 单个完整世界构建任务载体
type BuildTask struct {
	// BuildTaskConfig 构建任务基础配置参数
	BuildTaskConfig `task:"config"`
	// BuildTaskCheckpoint 构建任务断点存档数据
	BuildTaskCheckpoint `task:"checkpoint"`

	frame        define.Frame
	world        *building_world.BuildingWorld
	chunkManager *chunk_manager.ChunkManager
	blockBuilder *block_builder.BlockBuilder
	limiter      ratelimit.Limiter
	runMu        sync.Mutex
	runCtx       context.Context
	runCancel    context.CancelFunc
}

func (c *BuildTaskConfig) NewTask(frame define.Frame) define.Task {
	task := new(BuildTask)
	task.BuildTaskConfig = *c
	task.frame = frame
	return task
}
