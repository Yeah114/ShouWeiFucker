package build

func (b *BuildTask) Pause() error {
	b.cancelTask()
	return nil
}
