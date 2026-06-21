package utils

// Mod 返回非负取模结果。
func Mod(value, divisor int) int {
	result := value % divisor
	if result < 0 {
		result += divisor
	}
	return result
}

// ClampIndex 将索引约束到 [0, count-1] 范围内。
func ClampIndex(index, count int) int {
	return max(0, min(index, count-1))
}
