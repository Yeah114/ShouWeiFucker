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
	closeClient      bool
	listener         coreListener
	eventBus         EventBus.Bus
	tasks            []define.Task
	currentTaskIndex int
	config           FrameConfig
}

// FrameConfig 描述 Frame 的创建参数。
type FrameConfig struct {
	client.FrameConfig
	// Embedded 标记是否使用嵌入式运行模式，当前暂不参与逻辑。
	Embedded bool
}

// New 创建一个默认事件总线的 Frame。
func (c FrameConfig) New(coreClient *client.Client) *Frame {
	return &Frame{
		client:   coreClient,
		eventBus: EventBus.New(),
		config:   c,
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

// Connect 确保 Core 客户端已经连接。
func (f *Frame) Connect(ctx context.Context) error {
	if err := f.initClient(); err != nil {
		return fmt.Errorf("Frame.Connect: init client: %w", err)
	}
	if f.client == nil {
		return fmt.Errorf("Frame.Connect: nil client")
	}
	state, err := f.client.Frame().GetConnectionState(ctx)
	if err != nil {
		return fmt.Errorf("Frame.Connect: get connection state: %w", err)
	}
	if state.Connected {
		return nil
	}

	if _, err := f.client.Frame().StartConnection(ctx, f.config.FrameConfig); err != nil {
		return fmt.Errorf("Frame.Connect: start connection: %w", err)
	}

	state, err = f.client.Frame().GetConnectionState(ctx)
	if err != nil {
		return fmt.Errorf("Frame.Connect: get connection state after start: %w", err)
	}
	if !state.Connected {
		return fmt.Errorf("Frame.Connect: core disconnected after start: %s", state.CloseReason)
	}
	return nil
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

// Pause 暂停当前任务。
func (f *Frame) Pause() error {
	task := f.currentTask()
	if task == nil {
		return nil
	}
	if err := task.Pause(); err != nil {
		return fmt.Errorf("Frame.Pause: pause task %q: %w", task.Name(), err)
	}
	return nil
}

// Resume 恢复当前任务，并在其完成后继续执行剩余任务。
func (f *Frame) Resume() error {
	task := f.currentTask()
	if task == nil {
		return nil
	}
	if err := task.Resume(); err != nil {
		return fmt.Errorf("Frame.Resume: resume task %q: %w", task.Name(), err)
	}
	for i := f.currentTaskIndex + 1; i < len(f.tasks); i++ {
		f.currentTaskIndex = i
		task = f.tasks[i]
		if err := task.Start(); err != nil {
			return fmt.Errorf("Frame.Resume: start task %q: %w", task.Name(), err)
		}
	}
	return nil
}

// Stop 停止当前任务，并将当前任务索引重置为 0。
func (f *Frame) Stop() error {
	task := f.currentTask()
	if task == nil {
		f.currentTaskIndex = 0
		return nil
	}
	if err := task.Stop(); err != nil {
		return fmt.Errorf("Frame.Stop: stop task %q: %w", task.Name(), err)
	}
	f.currentTaskIndex = 0
	return nil
}

// Close 停止所有任务并关闭 Core 连接。
func (f *Frame) Close() error {
	if err := f.Stop(); err != nil {
		return fmt.Errorf("Frame.Close: %w", err)
	}
	if f.client != nil {
		if err := f.client.Frame().StopConnection(context.Background()); err != nil {
			return fmt.Errorf("Frame.Close: stop core connection: %w", err)
		}
	}
	if f.closeClient && f.client != nil {
		if err := f.client.Close(); err != nil {
			return fmt.Errorf("Frame.Close: close client: %w", err)
		}
	}
	f.client = nil
	if f.listener != nil {
		if err := f.listener.Close(); err != nil {
			return fmt.Errorf("Frame.Close: close listener: %w", err)
		}
		f.listener = nil
	}
	return nil
}

func (f *Frame) currentTask() define.Task {
	if f.currentTaskIndex < 0 || f.currentTaskIndex >= len(f.tasks) {
		return nil
	}
	return f.tasks[f.currentTaskIndex]
}

var _ define.Frame = (*Frame)(nil)
