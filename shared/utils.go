package shared

import "math"

func RootHeightFromWidth(width uint64) uint {
	return uint(math.Ceil(math.Log2(float64(width))))
}
