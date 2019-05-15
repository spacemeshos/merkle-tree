package merkle

import "fmt"

type Position struct {
	Index  uint64
	Height uint
}

func (p Position) String() string {
	return fmt.Sprintf("<h: %d i: %b>", p.Height, p.Index)
}

func (p Position) sibling() Position {
	return Position{
		Index:  p.Index ^ 1,
		Height: p.Height,
	}
}

func (p Position) isAncestorOf(other Position) bool {
	if p.Height < other.Height {
		return false
	}
	return p.Index == (other.Index >> (p.Height - other.Height))
}

func (p Position) isRightSibling() bool {
	return p.Index%2 == 1
}

func (p Position) parent() Position {
	return Position{
		Index:  p.Index >> 1,
		Height: p.Height + 1,
	}
}

func (p Position) leftChild() Position {
	return Position{
		Index:  p.Index << 1,
		Height: p.Height - 1,
	}
}

type positionsStack struct {
	positions []Position
}

func (s *positionsStack) Push(v Position) {
	s.positions = append(s.positions, v)
}

// Check the top of the stack for equality and pop the element if it's equal.
func (s *positionsStack) PopIfEqual(p Position) bool {
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
