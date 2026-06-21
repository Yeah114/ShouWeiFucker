package build

import "fmt"

// Resume 初始化任务并从断点继续执行构建任务。
func (b *BuildTask) Resume() error {
	if err := b.Init(); err != nil {
		return fmt.Errorf("BuildTask.Resume: %w", err)
	}
	ctx := b.startTaskContext()
	if err := b.run(ctx); err != nil {
		return fmt.Errorf("BuildTask.Resume: %w", err)
	}
	return nil
}
