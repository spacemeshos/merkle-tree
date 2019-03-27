package merkle

import (
	"errors"
	"sort"
)

var noMoreItems = errors.New("no more items")

type set map[uint64]bool

func (s set) asSortedSlice() []uint64 {
	var ret []uint64
	for key, value := range s {
		if value {
			ret = append(ret, key)
		}
	}
	sort.Slice(ret, func(i, j int) bool { return ret[i] < ret[j] })
	return ret
}

func setOf(members ...uint64) set {
	ret := make(set)
	for _, member := range members {
		ret[member] = true
	}
	return ret
}

type positionsIterator struct {
	s []uint64
}

func newPositionsIterator(positions set) *positionsIterator {
	s := positions.asSortedSlice()
	return &positionsIterator{s: s}
}

func (it *positionsIterator) peek() (pos position, found bool) {
	if len(it.s) == 0 {
		return position{}, false
	}
	index := it.s[0]
	return position{index: index}, true
}

// batchPop returns the indices of all positions up to endIndex.
func (it *positionsIterator) batchPop(endIndex uint64) set {
	res := make(set)
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

type leafIterator struct {
	indices []uint64
	leaves  [][]byte
}

// leafIterator.next() returns the leaf index and value
func (it *leafIterator) next() (position, []byte, error) {
	if len(it.indices) == 0 {
		return position{}, nil, noMoreItems
	}
	idx := it.indices[0]
	leaf := it.leaves[0]
	it.indices = it.indices[1:]
	it.leaves = it.leaves[1:]
	return position{index: idx}, leaf, nil
}

// leafIterator.peek() returns the leaf index but doesn't move the iterator to this leaf as next would do
func (it *leafIterator) peek() (position, []byte, error) {
	if len(it.indices) == 0 {
		return position{}, nil, noMoreItems
	}
	return position{index: it.indices[0]}, it.leaves[0], nil
}
