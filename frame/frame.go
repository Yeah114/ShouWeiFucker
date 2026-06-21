package frame

import (
	"context"
	"fmt"

	client "github.com/EmptyDea-Team/EmptyDea-core-client"
	"github.com/Yeah114/Fatalder/define"
	"github.com/asaskevich/EventBus"
)

// Frame 是 Fatalder 的运行框架实现，负责持有 Core 客户端、事件总线和任务列表。
type Frame struct {
	client           *client.Client
	eventBus         EventBus.Bus
	tasks            []define.Task
	currentTaskIndex int
}

// New 创建一个默认事件总线的 Frame。
func New(coreClient *client.Client) *Frame {
	return NewWithEventBus(coreClient, EventBus.New())
}

// NewWithEventBus 创建一个使用指定事件总线的 Frame。
func NewWithEventBus(coreClient *client.Client, bus EventBus.Bus) *Frame {
	return &Frame{
		client:   coreClient,
		eventBus: bus,
	}
}

// Client 返回底层 EmptyDea Core 客户端。
func (f *Frame) Client() *client.Client {
	return f.client
}

// EventBus 返回框架事件总线。
func (f *Frame) EventBus() EventBus.Bus {
	return f.eventBus
}

// Tasks 返回当前框架持有的任务列表。
func (f *Frame) Tasks() []define.Task {
	return f.tasks
}

// CurrentTaskIndex 返回当前正在处理的任务索引。
func (f *Frame) CurrentTaskIndex() int {
	return f.currentTaskIndex
}

// AddTask 添加任务到框架并返回自身，便于链式调用。
func (f *Frame) AddTask(task define.Task) define.Frame {
	f.tasks = append(f.tasks, task)
	return f
}

// Start 按添加顺序启动所有任务。
func (f *Frame) Start() error {
	for i, task := range f.tasks {
		f.currentTaskIndex = i
		if err := task.Start(); err != nil {
			return fmt.Errorf("Frame.Start: start task %q: %w", task.Name(), err)
		}
	}
	return nil
}

// Pause 暂停所有任务。
func (f *Frame) Pause() error {
	for i := len(f.tasks) - 1; i >= 0; i-- {
		f.currentTaskIndex = i
		task := f.tasks[i]
		if err := task.Pause(); err != nil {
			return fmt.Errorf("Frame.Pause: pause task %q: %w", task.Name(), err)
		}
	}
	return nil
}

// Resume 按添加顺序恢复所有任务。
func (f *Frame) Resume() error {
	for i, task := range f.tasks {
		f.currentTaskIndex = i
		if err := task.Resume(); err != nil {
			return fmt.Errorf("Frame.Resume: resume task %q: %w", task.Name(), err)
		}
	}
	return nil
}

// Stop 停止所有任务。
func (f *Frame) Stop() error {
	for i := len(f.tasks) - 1; i >= 0; i-- {
		f.currentTaskIndex = i
		task := f.tasks[i]
		if err := task.Stop(); err != nil {
			return fmt.Errorf("Frame.Stop: stop task %q: %w", task.Name(), err)
		}
	}
	return nil
}

// Close 停止所有任务并关闭 Core 连接。
func (f *Frame) Close() error {
	if err := f.Stop(); err != nil {
		return fmt.Errorf("Frame.Close: %w", err)
	}
	if f.client == nil {
		return nil
	}
	if err := f.client.Frame().StopConnection(context.Background()); err != nil {
		return fmt.Errorf("Frame.Close: stop core connection: %w", err)
	}
	return nil
}

var _ define.Frame = (*Frame)(nil)
