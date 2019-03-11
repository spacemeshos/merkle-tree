package merkle

import (
	"fmt"
	"math"
)

type position struct {
	index  uint64
	height uint
}

func (p position) String() string {
	return fmt.Sprintf("<h: %d i: %b>", p.height, p.index)
}

func (p position) sibling() position {
	return position{
		index:  p.index ^ 1,
		height: p.height,
	}
}

func (p position) isAncestorOf(other position) bool {
	if p.height < other.height {
		return false
	}
	return p.index == (other.index >> (p.height - other.height))
}

func (p position) isRightSibling() bool {
	return p.index%2 == 1
}

func (p position) parent() position {
	return position{
		index:  p.index >> 1,
		height: p.height + 1,
	}
}

func (p position) leftChild() position {
	return position{
		index:  p.index << 1,
		height: p.height - 1,
	}
}

func rootHeightFromWidth(width uint64) uint {
	return uint(math.Ceil(math.Log2(float64(width))))
}
