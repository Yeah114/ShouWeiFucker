package define

import (
	"github.com/EmptyDea-Team/EmptyDea-core-client"
	"github.com/asaskevich/EventBus"
)

type Frame interface {
	Client() *client.Client
	EventBus() EventBus.Bus
	// 所有任务
	Tasks() []Task
	// 当前任务索引
	CurrentTaskIndex() int
	// 添加任务
	AddTask(task Task) Frame
	// 运行所有任务
	Start() error
	Pause() error
	Resume() error
	// 停止所有任务
	Stop() error
	// 停止并关闭 Client
	Close() error
}

type Task interface {
	Name() string
	Frame() Frame
	Start() error
	Pause() error
	Resume() error
	Stop() error
}
