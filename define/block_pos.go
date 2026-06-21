package define

import "fmt"

// 为和国际版MC保持统一，世界范围被定义为 -64~319 ,现在网易的高度也是 -64 ~ 319 了，所以省事了不少
var WorldRange = Dimension(DimensionIDOverworld).Range()

// BlockPos holds the position of a block. The position is represented of an array with an x, y and z value,
// where the y value is positive.
type BlockPos [3]int

func (c BlockPos) SortStartAndEndPos(another BlockPos) (start, end BlockPos) {
	return SortStartAndEndPos(c, another)
}

func SortStartAndEndPos(s, e BlockPos) (start, end BlockPos) {
	if s.X() > e.X() {
		start[0] = e.X()
		end[0] = s.X()
	} else {
		start[0] = s.X()
		end[0] = e.X()
	}
	if s.Y() > e.Y() {
		start[1] = e.Y()
		end[1] = s.Y()
	} else {
		start[1] = s.Y()
		end[1] = e.Y()
	}
	if s.Z() > e.Z() {
		start[2] = e.Z()
		end[2] = s.Z()
	} else {
		start[2] = s.Z()
		end[2] = e.Z()
	}
	return start, end
}

func (c BlockPos) BlockSize(another BlockPos) Size {
	return BlockSize(c, another)
}

func BlockSize(start, end BlockPos) Size {
	size := end.Sub(start)
	if size[0] < 0 {
		size[0] = -size[0]
	}
	if size[1] < 0 {
		size[1] = -size[1]
	}
	if size[2] < 0 {
		size[2] = -size[2]
	}
	return Size{
		Width:  size[0],
		Height: size[1],
		Length: size[2],
	}
}

func (p BlockPos) OutOfYBounds() bool {
	y := p[1]
	return y > WorldRange[1] || y < WorldRange[0]
}

// String converts the Pos to a string in the format (1,2,3) and returns it.
func (p BlockPos) String() string {
	return fmt.Sprintf("(%v,%v,%v)", p[0], p[1], p[2])
}

func (p BlockPos) Sub(po BlockPos) (offset BlockPos) {
	offset[0] = p[0] - po[0]
	offset[1] = p[1] - po[1]
	offset[2] = p[2] - po[2]
	return offset
}

func (p BlockPos) Add(po BlockPos) (offset BlockPos) {
	offset[0] = p[0] + po[0]
	offset[1] = p[1] + po[1]
	offset[2] = p[2] + po[2]
	return offset
}

// X returns the X coordinate of the block position.
func (p BlockPos) X() int {
	return p[0]
}

// Y returns the Y coordinate of the block position.
func (p BlockPos) Y() int {
	return p[1]
}

// Z returns the Z coordinate of the block position.
func (p BlockPos) Z() int {
	return p[2]
}

func GetPosFromNBT(nbt map[string]interface{}) (x, y, z int, success bool) {
	if ax, hasK := nbt["x"]; hasK {
		if cx, success := ax.(int32); success {
			x = int(cx)
		} else {
			return 0, 0, 0, false
		}
	} else {
		return 0, 0, 0, false
	}
	if ay, hasK := nbt["y"]; hasK {
		if cy, success := ay.(int32); success {
			y = int(cy)
		} else {
			return 0, 0, 0, false
		}
	} else {
		return 0, 0, 0, false
	}
	if az, hasK := nbt["z"]; hasK {
		if cz, success := az.(int32); success {
			z = int(cz)
		} else {
			return 0, 0, 0, false
		}
	} else {
		return 0, 0, 0, false
	}
	return x, y, z, true
}

func GetBlockPosFromNBT(nbt map[string]interface{}) (p BlockPos, success bool) {
	if x, y, z, success := GetPosFromNBT(nbt); success {
		return BlockPos{x, y, z}, true
	} else {
		return BlockPos{0, 0, 0}, false
	}
}
