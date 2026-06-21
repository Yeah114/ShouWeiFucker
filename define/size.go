package define

type Size struct {
	Width, Height, Length int
}

func (s *Size) Volume() int {
	return s.Width * s.Height * s.Length
}

func (s *Size) ChunkXCount() int {
	return (s.Width + 16 - 1) / 16
}

func (s *Size) ChunkZCount() int {
	return (s.Length + 16 - 1) / 16
}

func (s *Size) ChunkCount() int {
	return s.ChunkXCount() * s.ChunkZCount()
}
