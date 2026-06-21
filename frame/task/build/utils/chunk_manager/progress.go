package chunk_manager

// Progress 返回当前已处理的组数以及总组数。
func (c *ChunkManager) Progress() (int, int) {
	return c.progress, c.max
}
