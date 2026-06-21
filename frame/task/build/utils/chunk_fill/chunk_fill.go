package chunk_fill

import (
	"fmt"

	"github.com/EmptyDea-Team/bedrock-world-operator/block"
	"github.com/EmptyDea-Team/bedrock-world-operator/chunk"
	"github.com/Yeah114/Fatalder/define"
	"github.com/Yeah114/Fatalder/utils"
)

// Bedrock 的 fill 存在体积上限。
// 用户要求“不能超过 32767”，因此这里严格使用 32766 作为安全上限，
// 避免任何边界值在不同服务端/转发层中产生兼容性问题。
const maxFillVolume = 32766

// blockInfo 是运行时 ID 解码后的轻量缓存结果。
// 这里缓存 name/state 的目的是避免在大体量结构生成命令时反复做
// RuntimeIDToState 转换，减少 CPU 消耗。
type blockInfo struct {
	name  string
	state string
	isAir bool
}

// planeRect 表示单个 Y 层上的一个二维矩形。
//
// 注意它不是最终输出命令，而是“层内合并”的中间结果：
// 1. 先在当前 y 层把连续同材质方块合成一个 XZ 平面矩形。
// 2. 再尝试把这个矩形沿 Y 方向继续向上叠高。
// 3. 最后才决定发 fill 还是 setblock。
type planeRect struct {
	x     int
	z     int
	width int
	depth int
	id    uint32
}

// cuboid 表示最终准备输出的三维长方体。
// 它可能直接映射成一条 fill，也可能因为体积超限被继续拆分。
type cuboid struct {
	x      int
	y      int
	z      int
	width  int
	height int
	depth  int
}

// GenerateChunksCommand 会把多个区块展平成一个统一的三维网格，然后生成一串
// fill/setblock 命令来重建该区域。
//
// 整体算法分四步：
// 1. 先计算输入区块的整体包围盒，把所有 block 映射到同一个连续网格中。
// 2. 在每个 y 层上，从左到右、从前到后扫描，寻找“以当前点为起点”的最大矩形。
// 3. 对该矩形继续沿 y 方向叠高，得到尽量大的长方体。
// 4. 若长方体体积超过 maxFillVolume，则按最少拆分段数原则继续切开。
//
// 这个版本不是理论上的全局最优覆盖，但它有几个实际优点：
// - 实现足够直接，便于维护和调试
// - 输出稳定，同一输入通常得到一致命令序
// - 命令数明显优于逐点 setblock
// - 相比更激进的全局拼接算法，CPU 成本更容易控制
func GenerateChunksCommand(table *block.BlockRuntimeIDTable, chunks map[define.ChunkPos]*chunk.Chunk, startPos define.BlockPos) <-chan string {
	ch := make(chan string)

	go func() {
		defer close(ch)

		if table == nil || len(chunks) == 0 {
			return
		}

		type chunkEntry struct {
			pos   define.ChunkPos
			chunk *chunk.Chunk
		}

		entries := make([]chunkEntry, 0, len(chunks))
		for pos, c := range chunks {
			if c == nil {
				continue
			}
			subChunks := c.Sub()
			if len(subChunks) == 0 {
				continue
			}
			entries = append(entries, chunkEntry{pos: pos, chunk: c})
		}

		if len(entries) == 0 {
			return
		}

		minChunkX := entries[0].pos.X()
		maxChunkX := minChunkX
		minChunkZ := entries[0].pos.Z()
		maxChunkZ := minChunkZ
		minY := entries[0].chunk.Range().Min()
		maxY := entries[0].chunk.Range().Min() + len(entries[0].chunk.Sub())*16 - 1

		for _, entry := range entries[1:] {
			pos := entry.pos
			c := entry.chunk
			if pos.X() < minChunkX {
				minChunkX = pos.X()
			}
			if pos.X() > maxChunkX {
				maxChunkX = pos.X()
			}
			if pos.Z() < minChunkZ {
				minChunkZ = pos.Z()
			}
			if pos.Z() > maxChunkZ {
				maxChunkZ = pos.Z()
			}

			chunkMinY := c.Range().Min()
			chunkMaxY := chunkMinY + len(c.Sub())*16 - 1
			if chunkMinY < minY {
				minY = chunkMinY
			}
			if chunkMaxY > maxY {
				maxY = chunkMaxY
			}
		}

		sizeX := int(maxChunkX-minChunkX+1) * 16
		sizeZ := int(maxChunkZ-minChunkZ+1) * 16
		sizeY := maxY - minY + 1
		if sizeX <= 0 || sizeY <= 0 || sizeZ <= 0 {
			return
		}

		// 这里把整个待处理区域抽象成一个稠密三维数组。
		//
		// blockIDs:
		//   保存每个位置的 runtime ID。
		// canSet:
		//   表示该位置是否存在“有效输入块且尚未被消费”。
		//
		// 之所以不用 map，而用一维切片扁平化存储：
		// - 顺序扫描时局部性更好
		// - 下标计算比多层 map 更稳定
		// - 后续大量 canUse/getIndex 判断成本更低
		total := sizeX * sizeY * sizeZ
		blockIDs := make([]uint32, total)
		canSet := make([]bool, total)

		// 扁平化索引。
		// 布局顺序是 X -> Y -> Z。
		// 这里不追求“数学上最自然”，而是保持一致即可。
		getIndex := func(x, y, z int) int {
			return ((x*sizeY)+y)*sizeZ + z
		}

		// 把输入 chunk 的局部坐标系，复制到统一全局网格中。
		// 复制完以后，后续算法完全不再关心“这些块原来属于哪个 chunk”，
		// 只把它们当成一个大的连续立方体区域来处理。
		for _, entry := range entries {
			pos := entry.pos
			c := entry.chunk
			chunkMinY := c.Range().Min()
			subChunks := c.Sub()
			height := len(subChunks) * 16
			if height == 0 {
				continue
			}

			offsetX := int(pos.X()-minChunkX) * 16
			offsetZ := int(pos.Z()-minChunkZ) * 16
			offsetY := chunkMinY - minY

			for x := 0; x < 16; x++ {
				for y := 0; y < height; y++ {
					worldY := chunkMinY + y
					gy := offsetY + y
					if gy < 0 || gy >= sizeY {
						continue
					}
					for z := 0; z < 16; z++ {
						gx := offsetX + x
						gz := offsetZ + z
						if gx < 0 || gx >= sizeX || gz < 0 || gz >= sizeZ {
							continue
						}
						idx := getIndex(gx, gy, gz)
						blockIDs[idx] = c.Block(uint8(x), int16(worldY), uint8(z), 0)
						canSet[idx] = true
					}
				}
			}
		}

		cache := make(map[uint32]blockInfo)
		baseX := int(startPos.X())
		baseY := int(startPos.Y())
		baseZ := int(startPos.Z())

		// 向上取整除法，用于估算拆分段数。
		ceilDiv := func(a, b int) int {
			if b <= 0 {
				return a
			}
			if a <= 0 {
				return 0
			}
			return (a + b - 1) / b
		}

		// runtime ID -> 具体方块信息。
		//
		// 这里有两个重要约定：
		// 1. air 统一视为“无需输出命令”的块。
		// 2. 无法识别的 runtime ID 退化为空气，保证算法不会生成非法命令。
		getBlockInfo := func(id uint32) blockInfo {
			if info, ok := cache[id]; ok {
				return info
			}

			airRuntimeID := table.AirRuntimeID()
			if id == airRuntimeID {
				info := blockInfo{name: "minecraft:air", state: "[]", isAir: true}
				cache[id] = info
				return info
			}

			name, properties, found := table.RuntimeIDToState(id)
			state := utils.PropertiesToStateStr(properties)
			if !found {
				name = "minecraft:air"
				state = "[]"
			}

			info := blockInfo{name: name, state: state, isAir: id == airRuntimeID || name == "minecraft:air"}
			cache[id] = info
			return info
		}

		// sendFill/sendSetBlock 是唯一命令出口。
		// 所有上层逻辑只负责决定“发多大、多高、多宽的几何体”，
		// 不直接拼接命令字符串。
		sendFill := func(x1, y1, z1, x2, y2, z2 int, info blockInfo) {
			cmd := fmt.Sprintf(
				"fill %d %d %d %d %d %d %s %s\n",
				baseX+x1,
				baseY+y1,
				baseZ+z1,
				baseX+x2,
				baseY+y2,
				baseZ+z2,
				info.name,
				info.state,
			)
			ch <- cmd
		}

		sendSetBlock := func(x, y, z int, info blockInfo) {
			cmd := fmt.Sprintf(
				"setblock %d %d %d %s %s\n",
				baseX+x,
				baseY+y,
				baseZ+z,
				info.name,
				info.state,
			)
			ch <- cmd
		}

		var emitCuboid func(box cuboid, info blockInfo)
		// emitCuboid 负责把一个三维长方体安全地输出成命令。
		//
		// 规则固定为：
		// - 体积 <= 0：忽略
		// - 体积 == 1：发 setblock
		// - 1 < 体积 <= maxFillVolume：发一条 fill
		// - 体积超限：继续拆分，直到每个子块都满足 fill 上限
		//
		// 之所以把拆分收口到这里，是为了保证上游所有搜索逻辑都不必重复
		// 处理“万一超限怎么办”的细节。
		emitCuboid = func(box cuboid, info blockInfo) {
			volume := box.width * box.height * box.depth
			if volume <= 0 {
				return
			}
			if volume == 1 {
				sendSetBlock(box.x, box.y, box.z, info)
				return
			}
			if volume <= maxFillVolume {
				sendFill(
					box.x,
					box.y,
					box.z,
					box.x+box.width-1,
					box.y+box.height-1,
					box.z+box.depth-1,
					info,
				)
				return
			}

			type splitPlan struct {
				axis        int
				maxAxisSize int
				parts       int
				priority    int
			}

			// 尝试直接找到“沿某一个轴切开即可满足体积限制”的最优方案。
			//
			// 对于每个候选轴：
			// - otherVolumes 表示另外两个维度乘积
			// - maxAxisSize 表示在当前另外两维固定时，该轴最多允许多长
			// - parts 表示需要切成几段
			//
			// 优先级 priority 的设计：
			// - Y 优先级最高（0），尽量保留竖向整体性
			// - X 次之
			// - Z 最后
			//
			// 原因是建筑结构里很多层叠特征本来就在 Y 上，优先保 Y
			// 一般更符合直觉，也更稳定。
			plans := make([]splitPlan, 0, 3)
			dims := [3]int{box.width, box.height, box.depth}
			otherVolumes := [3]int{
				box.height * box.depth,
				box.width * box.depth,
				box.width * box.height,
			}
			priorities := [3]int{1, 0, 2}

			for axis := range dims {
				if dims[axis] <= 1 || otherVolumes[axis] <= 0 || otherVolumes[axis] > maxFillVolume {
					continue
				}
				maxAxisSize := maxFillVolume / otherVolumes[axis]
				if maxAxisSize <= 0 {
					continue
				}
				plans = append(plans, splitPlan{
					axis:        axis,
					maxAxisSize: maxAxisSize,
					parts:       ceilDiv(dims[axis], maxAxisSize),
					priority:    priorities[axis],
				})
			}

			if len(plans) > 0 {
				// 先选最少分段数，再按 priority 决定同分时的拆分轴。
				best := plans[0]
				for _, plan := range plans[1:] {
					if plan.parts < best.parts || (plan.parts == best.parts && plan.priority < best.priority) {
						best = plan
					}
				}

				for offset := 0; offset < dims[best.axis]; offset += best.maxAxisSize {
					segment := best.maxAxisSize
					if remain := dims[best.axis] - offset; remain < segment {
						segment = remain
					}
					part := box
					switch best.axis {
					case 0:
						part.x += offset
						part.width = segment
					case 1:
						part.y += offset
						part.height = segment
					default:
						part.z += offset
						part.depth = segment
					}
					emitCuboid(part, info)
				}
				return
			}

			// 理论上很少走到这里。
			// 这里只有在“单轴精确限长切分”也不好直接处理时，做一个简单的二分递归兜底。
			// 兜底策略依然偏向保 Y，再在 X/Z 中选较长边。
			axis := 1
			if box.width > box.height || (box.width == box.height && box.width >= box.depth) {
				axis = 0
			}
			if box.depth > dims[axis] || (box.depth == dims[axis] && axis == 0) {
				axis = 2
			}

			first := box
			second := box
			switch axis {
			case 0:
				split := box.width / 2
				if split <= 0 {
					split = 1
				}
				first.width = split
				second.x += split
				second.width -= split
			case 1:
				split := box.height / 2
				if split <= 0 {
					split = 1
				}
				first.height = split
				second.y += split
				second.height -= split
			default:
				split := box.depth / 2
				if split <= 0 {
					split = 1
				}
				first.depth = split
				second.z += split
				second.depth -= split
			}

			emitCuboid(first, info)
			emitCuboid(second, info)
		}

		// canUse 表示某个位置当前是否仍可被当前候选几何体使用。
		//
		// 它同时检查：
		// - 是否越界
		// - 是否还没被前面的矩形/长方体消费掉
		// - runtime ID 是否完全一致
		//
		// 这是整个合并算法最核心的判定之一。
		canUse := func(x, y, z int, id uint32) bool {
			if x < 0 || x >= sizeX || y < 0 || y >= sizeY || z < 0 || z >= sizeZ {
				return false
			}
			idx := getIndex(x, y, z)
			return canSet[idx] && blockIDs[idx] == id
		}

		// findBestRect 会在固定 y 层上，以 (x, z) 为左上起点，寻找一个“局部最优矩形”。
		//
		// 它做了两次搜索：
		// 1. 固定“逐层加深 depth”，每增加一行就更新当前最小 width
		// 2. 固定“逐列加宽 width”，每增加一列就更新当前最小 depth
		//
		// 为什么要做两次：
		// - 只做单方向扩展会被扫描方向偏置
		// - 某些结构适合“先宽后深”，另一些适合“先深后宽”
		// - 两次都算一遍，再选面积更大的那个，能降低局部贪心误差
		//
		// 注意这里故意只做“局部最好”，不是全局最优覆盖。
		// 原因是全局最优矩形覆盖复杂度太高，维护成本也高，不符合当前 KISS 目标。
		findBestRect := func(x, y, z int, blockID uint32) planeRect {
			best := planeRect{
				x:     x,
				z:     z,
				width: 1,
				depth: 1,
				id:    blockID,
			}
			bestArea := 0

			// updateBest 的 tie-break 规则：
			// 1. 优先面积更大
			// 2. 面积相同时优先 width 更大
			// 3. 还相同时优先 depth 更大
			//
			// 这样做的好处是：
			// - 同面积下尽量减少“细碎窄条”
			// - 命令形状更规整
			// - 输出更加稳定
			updateBest := func(width, depth int) {
				if width <= 0 || depth <= 0 {
					return
				}
				area := width * depth
				if area > bestArea || (area == bestArea && (width > best.width || (width == best.width && depth > best.depth))) {
					best = planeRect{
						x:     x,
						z:     z,
						width: width,
						depth: depth,
						id:    blockID,
					}
					bestArea = area
				}
			}

			// 第一轮：以“向 Z 方向加深”为主轴扩展。
			// 每扩一层，都重新计算这一层在 X 方向还能保持多宽。
			currentWidth := 0
			for depth := 0; z+depth < sizeZ; depth++ {
				rowWidth := 0
				for xx := x; xx < sizeX; xx++ {
					if !canUse(xx, y, z+depth, blockID) {
						break
					}
					rowWidth++
				}
				if rowWidth == 0 {
					break
				}
				if depth == 0 || rowWidth < currentWidth {
					currentWidth = rowWidth
				}

				depthLen := depth + 1
				widthLimit := maxFillVolume / depthLen
				width := currentWidth
				if width > widthLimit {
					width = widthLimit
				}
				updateBest(width, depthLen)
			}

			// 第二轮：以“向 X 方向加宽”为主轴扩展。
			// 每扩一列，都重新计算这一列在 Z 方向还能保持多深。
			currentDepth := 0
			for width := 0; x+width < sizeX; width++ {
				columnDepth := 0
				for zz := z; zz < sizeZ; zz++ {
					if !canUse(x+width, y, zz, blockID) {
						break
					}
					columnDepth++
				}
				if columnDepth == 0 {
					break
				}
				if width == 0 || columnDepth < currentDepth {
					currentDepth = columnDepth
				}

				widthLen := width + 1
				depthLimit := maxFillVolume / widthLen
				depth := currentDepth
				if depth > depthLimit {
					depth = depthLimit
				}
				updateBest(widthLen, depth)
			}

			return best
		}

		// canStackRect 判断一个已经在当前 y 层找到的矩形，能否原样复制到更高一层。
		//
		// 这里要求非常严格：
		// - footprint（XZ 投影）必须完全一致
		// - 矩形覆盖的每个点都还是同一个 runtime ID
		// - 中间不能掺杂任何已经被消费的块
		//
		// 这种严格约束虽然保守，但可以保证生成的 fill 一定正确。
		canStackRect := func(rect planeRect, y int) bool {
			for xx := rect.x; xx < rect.x+rect.width; xx++ {
				for zz := rect.z; zz < rect.z+rect.depth; zz++ {
					if !canUse(xx, y, zz, rect.id) {
						return false
					}
				}
			}
			return true
		}

		// markConsumed 把已经并入某个长方体的所有块标记为“已处理”。
		//
		// 这一步非常关键：
		// - 它保证同一个方块只会被输出一次
		// - 后续扫描遇到这些位置时会直接跳过
		// - 整个算法因此能保持单次覆盖，不会重复发命令
		markConsumed := func(box cuboid) {
			for xx := box.x; xx < box.x+box.width; xx++ {
				for yy := box.y; yy < box.y+box.height; yy++ {
					for zz := box.z; zz < box.z+box.depth; zz++ {
						canSet[getIndex(xx, yy, zz)] = false
					}
				}
			}
		}

		// 主扫描顺序固定为 y -> x -> z。
		//
		// 理解方式可以是：
		// - 先按层处理
		// - 在一层里从左上角向右下角扫
		// - 每遇到一个还没处理的非空气块，就以它为 seed 找一个局部最大矩形
		// - 再把这个矩形尽可能向上叠高
		//
		// 这是一种经典的“扫描线 + 消费标记”思路。
		for y := 0; y < sizeY; y++ {
			for x := 0; x < sizeX; x++ {
				for z := 0; z < sizeZ; z++ {
					idx := getIndex(x, y, z)
					if !canSet[idx] {
						continue
					}

					// 对空气的处理很刻意：
					// - 不生成任何命令
					// - 但要立刻标记为 false，避免后续重复检查
					blockID := blockIDs[idx]
					info := getBlockInfo(blockID)
					if info.isAir {
						canSet[idx] = false
						continue
					}

					// 第一步：在当前层上找二维矩形。
					rect := findBestRect(x, y, z, blockID)
					area := rect.width * rect.depth

					// 第二步：根据体积上限，计算这个矩形理论上最多能叠多高。
					// 例如面积是 100，那么高度上限就是 32766 / 100。
					maxHeight := maxFillVolume / area
					if maxHeight <= 0 {
						maxHeight = 1
					}

					// 第三步：在不超过 maxHeight 的前提下，尽量向上叠层。
					height := 1
					for nextY := y + 1; nextY < sizeY && height < maxHeight; nextY++ {
						if !canStackRect(rect, nextY) {
							break
						}
						height++
					}

					// 第四步：消费掉这个长方体，并统一交给 emitCuboid 输出命令。
					box := cuboid{
						x:      rect.x,
						y:      y,
						z:      rect.z,
						width:  rect.width,
						height: height,
						depth:  rect.depth,
					}
					markConsumed(box)
					emitCuboid(box, info)
				}
			}
		}
	}()

	return ch
}
