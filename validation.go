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
	layer := uint(0)
	idx, activeNode, err := v.leaves.next()
	if err != nil {
		return nil, err
	}
	var lChild, rChild, sibling []byte
	for {
		if layer == stopAtLayer {
			break
		}
		if v.shouldCalcSubtree(idx, layer) {
			sibling, err = v.calcRoot(layer)
			if err != nil {
				return nil, err
			}
		} else {
			var err error
			sibling, err = v.proofNodes.next()
			if err == noMoreItems {
				break
			}
		}
		if isRightNode(idx, layer) {
			lChild, rChild = sibling, activeNode
		} else {
			lChild, rChild = activeNode, sibling
		}
		activeNode = v.hash(lChild, rChild)
		layer++
	}
	return activeNode, nil
}

func isRightNode(idx uint64, layer uint) bool {
	return (idx>>layer)%2 == 1
}

// shouldCalcSubtree returns true if the paths to idx (current leaf) and the nextIdx (next one) diverge at the current
// layer, so the next sibling should be the root of the subtree to the right.
func (v *validator) shouldCalcSubtree(idx uint64, layer uint) bool {
	nextIdx, _, err := v.leaves.peek()
	if err == noMoreItems {
		return false
	}
	// When eliminating the `layer` least significant bits of the bitwise xor of the current and next leaf index we
	// expect to get 1 at the divergence point.
	return (idx^nextIdx)>>layer == 1
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
func (it *leafIterator) next() (uint64, []byte, error) {
	if len(it.indices) == 0 {
		return 0, nil, noMoreItems
	}
	idx := it.indices[0]
	leaf := it.leaves[0]
	it.indices = it.indices[1:]
	it.leaves = it.leaves[1:]
	return idx, leaf, nil
}

// leafIterator.peek() returns the leaf index but doesn't move the iterator to this leaf as next would do
func (it *leafIterator) peek() (uint64, []byte, error) {
	if len(it.indices) == 0 {
		return 0, nil, noMoreItems
	}
	return it.indices[0], it.leaves[0], nil
}
