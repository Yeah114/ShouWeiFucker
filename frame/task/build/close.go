package build

func (b *BuildTask) Close() error {
	b.cancelTask()
	return nil
}
