package merkle

import (
	"bytes"
	"errors"
	"fmt"
)

const MaxUint = ^uint(0)

// ValidatePartialTree uses leafIndices, leaves and proof to calculate the merkle root of the tree and then compares it
// to expectedRoot.
func ValidatePartialTree(leafIndices []uint64, leaves, proof [][]byte, expectedRoot []byte,
	hash func(lChild, rChild []byte) []byte) (bool, error) {
	v, err := newValidator(leafIndices, leaves, proof, hash)
	if err != nil {
		return false, err
	}
	root, err := v.calcRoot(MaxUint)
	return bytes.Equal(root, expectedRoot), err
}

func newValidator(leafIndices []uint64, leaves, proof [][]byte,
	hash func(lChild, rChild []byte) []byte) (validator, error) {
	if len(leafIndices) != len(leaves) {
		return validator{}, fmt.Errorf("number of leaves (%d) must equal number of indices (%d)", len(leaves),
			len(leafIndices))
	}
	if len(leaves) == 0 {
		return validator{}, fmt.Errorf("at least one leaf is required for validation")
	}
	proofNodes := &proofIterator{proof}
	leafIt := &leafIterator{leafIndices, leaves}

	return validator{leaves: leafIt, proofNodes: proofNodes, hash: hash}, nil
}

type validator struct {
	leaves     *leafIterator
	proofNodes *proofIterator
	hash       func(lChild, rChild []byte) []byte
}

func (v *validator) calcRoot(stopAtLayer uint) ([]byte, error) {
	activePos, activeNode, err := v.leaves.next()
	if err != nil {
		return nil, err
	}
	var lChild, rChild, sibling []byte
	for {
		if activePos.height == stopAtLayer {
			break
		}
		nextLeafPos, _, err := v.leaves.peek()
		if err == nil && activePos.sibling().isAncestorOf(nextLeafPos) {
			sibling, err = v.calcRoot(activePos.height)
			if err != nil {
				return nil, err
			}
		} else {
			sibling, err = v.proofNodes.next()
			if err == noMoreItems {
				break
			}
		}
		if activePos.isRightSibling() {
			lChild, rChild = sibling, activeNode
		} else {
			lChild, rChild = activeNode, sibling
		}
		activeNode = v.hash(lChild, rChild)
		activePos = activePos.parent()
	}
	return activeNode, nil
}

var noMoreItems = errors.New("no more items")

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

type position struct {
	index  uint64
	height uint
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
