package build

const (
	// EventNameInitStart 初始化开始事件。
	EventNameInitStart = Name + ".Init.Start"
	// EventNameInitOpenWorld 开始打开源世界事件。
	EventNameInitOpenWorld = Name + ".Init.OpenWorld"
	// EventNameInitFinish 初始化完成事件。
	EventNameInitFinish = Name + ".Init.Finish"

	// EventNameRunStart 构建主流程开始事件。
	EventNameRunStart = Name + ".Run.Start"
	// EventNameRunChunkGroupStart 区块组开始处理事件。
	EventNameRunChunkGroupStart = Name + ".Run.ChunkGroup.Start"
	// EventNameRunChunkGroupLoaded 区块组数据读取完成事件。
	EventNameRunChunkGroupLoaded = Name + ".Run.ChunkGroup.Loaded"
	// EventNameRunCommandsGenerated 区块组构建命令生成完成事件。
	EventNameRunCommandsGenerated = Name + ".Run.Commands.Generated"
	// EventNameRunCommandSent 单条构建命令发送完成事件。
	EventNameRunCommandSent = Name + ".Run.Command.Sent"
	// EventNameRunChunkGroupFinish 区块组处理完成事件。
	EventNameRunChunkGroupFinish = Name + ".Run.ChunkGroup.Finish"
	// EventNameRunFinish 构建主流程完成事件。
	EventNameRunFinish = Name + ".Run.Finish"
)

// publish 向任务所属框架发布构建事件。
func (b *BuildTask) publish(name string, args ...any) {
	b.frame.EventBus().Publish(name, args...)
}
