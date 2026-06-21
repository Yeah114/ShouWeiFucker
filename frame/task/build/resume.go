package build

import "fmt"

func (b *BuildTask) Resume() error {
	if err := b.Init(); err != nil {
		return fmt.Errorf("BuildTask.Resume: %w", err)
	}
	return nil
}
