package merkle

import (
	"errors"
	"sort"
)

var noMoreItems = errors.New("no more items")

type Set map[uint64]bool

func (s Set) AsSortedSlice() []uint64 {
	var ret []uint64
	for key, value := range s {
		if value {
			ret = append(ret, key)
		}
	}
	sort.Slice(ret, func(i, j int) bool { return ret[i] < ret[j] })
	return ret
}

func SetOf(members ...uint64) Set {
	ret := make(Set)
	for _, member := range members {
		ret[member] = true
	}
	return ret
}

type positionsIterator struct {
	s []uint64
}

func NewPositionsIterator(positions Set) *positionsIterator {
	s := positions.AsSortedSlice()
	return &positionsIterator{s: s}
}

func (it *positionsIterator) peek() (pos Position, found bool) {
	if len(it.s) == 0 {
		return Position{}, false
	}
	index := it.s[0]
	return Position{Index: index}, true
}

// batchPop returns the indices of all positions up to endIndex.
func (it *positionsIterator) batchPop(endIndex uint64) Set {
	res := make(Set)
	for len(it.s) > 0 && it.s[0] < endIndex {
		res[it.s[0]] = true
		it.s = it.s[1:]
	}
	return res
}

type proofIterator struct {
	nodes [][]byte
}

func (it *proofIterator) next() ([]byte, error) {
	if len(it.nodes) == 0 {
		return nil, noMoreItems
	}
	n := it.nodes[0]
	it.nodes = it.nodes[1:]
	return n, nil
}

type LeafIterator struct {
	indices []uint64
	leaves  [][]byte
}

// LeafIterator.next() returns the leaf index and value
func (it *LeafIterator) next() (Position, []byte, error) {
	if len(it.indices) == 0 {
		return Position{}, nil, noMoreItems
	}
	idx := it.indices[0]
	leaf := it.leaves[0]
	it.indices = it.indices[1:]
	it.leaves = it.leaves[1:]
	return Position{Index: idx}, leaf, nil
}

// LeafIterator.peek() returns the leaf index but doesn't move the iterator to this leaf as next would do
func (it *LeafIterator) peek() (Position, []byte, error) {
	if len(it.indices) == 0 {
		return Position{}, nil, noMoreItems
	}
	return Position{Index: it.indices[0]}, it.leaves[0], nil
}
