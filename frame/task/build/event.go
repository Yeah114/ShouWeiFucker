package build

const (
	// EventNameInitStart 初始化开始事件。
	EventNameInitStart = Name + ".Init.Start"
	// EventNameInitOpenWorld 开始打开源世界事件。
	EventNameInitOpenWorld = Name + ".Init.OpenWorld"
	// EventNameInitFinish 初始化完成事件。
	EventNameInitFinish = Name + ".Init.Finish"
)

// publish 向任务所属框架发布构建事件。
func (b *BuildTask) publish(name string, args ...any) {
	b.frame.EventBus().Publish(name, args...)
}
