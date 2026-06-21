package build

import "fmt"

// Start 初始化任务并从当前断点开始执行构建任务。
func (b *BuildTask) Start() error {
	if err := b.Init(); err != nil {
		return fmt.Errorf("BuildTask.Start: %w", err)
	}
	ctx := b.startTaskContext()
	if err := b.run(ctx); err != nil {
		return fmt.Errorf("BuildTask.Start: %w", err)
	}
	return nil
}
