package chunk_manager

// Progress 返回当前已处理的组数以及总组数。
func (c *ChunkManager) Progress() (int, int) {
	return c.progress, c.max
}

// SetProgress 设置当前已处理的组数，用于中断后回退到未完成的区块组。
func (c *ChunkManager) SetProgress(progress int) {
	if progress < 0 {
		progress = 0
	}
	if progress > c.max {
		progress = c.max
	}
	c.progress = progress
}
