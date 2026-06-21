package build

func (b *BuildTask) Pause() error {
	b.cancelRun()
	return nil
}
