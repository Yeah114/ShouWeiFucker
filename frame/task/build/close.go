package build

func (b *BuildTask) Close() error {
	b.cancelRun()
	return nil
}
