package merkle

import "fmt"

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

type positionsStack struct {
	positions []position
}

func (s *positionsStack) Push(v position) {
	s.positions = append(s.positions, v)
}

// Check the top of the stack for equality and pop the element if it's equal.
func (s *positionsStack) PopIfEqual(p position) bool {
	l := len(s.positions)
	if l == 0 {
		return false
	}
	if s.positions[l-1] == p {
		s.positions = s.positions[:l-1]
		return true
	}
	return false
}
