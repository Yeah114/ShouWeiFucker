package build

import (
	"context"
	"fmt"
)

// Resume 初始化任务并从断点继续执行构建任务。
func (b *BuildTask) Resume() error {
	if err := b.Init(); err != nil {
		return fmt.Errorf("BuildTask.Resume: %w", err)
	}
	if err := b.run(context.Background()); err != nil {
		return fmt.Errorf("BuildTask.Resume: %w", err)
	}
	return nil
}
