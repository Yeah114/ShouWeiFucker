package build

func (b *BuildTask) Stop() error {
	b.cancelTask()
	return nil
}
