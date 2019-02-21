package merkle

import (
	"bytes"
	"errors"
	"fmt"
)

const MaxUint = ^uint(0)

// ValidatePartialTree uses leafIndices, leaves and proof to calculate the merkle root of the tree and then compares it
// to expectedRoot.
func ValidatePartialTree(leafIndices []uint64, leaves, proof []Node, expectedRoot Node) (bool, error) {
	v, err := newValidator(leafIndices, leaves, proof)
	if err != nil {
		return false, err
	}
	root := v.calcRoot(MaxUint)
	return bytes.Equal(root, expectedRoot), nil
}

func newValidator(leafIndices []uint64, leaves, proof []Node) (validator, error) {
	if len(leafIndices) != len(leaves) {
		return validator{}, fmt.Errorf("number of leaves (%d) must equal number of indices (%d)", len(leaves), len(leafIndices))
	}
	if len(leaves) == 0 {
		return validator{}, fmt.Errorf("at least one leaf is required for validation")
	}
	if len(leaves)+len(proof) == 1 {
		return validator{}, fmt.Errorf("tree of size 1 not supported")
	}
	proofNodes := &proofIterator{proof}
	leafIt := &leafIterator{leafIndices, leaves}

	return validator{leafIt, proofNodes}, nil
}

type validator struct {
	leaves     *leafIterator
	proofNodes *proofIterator
}

func (v *validator) calcRoot(stopAtLayer uint) Node {
	layer := uint(0)
	idx, activeNode, err := v.leaves.next()
	if err != nil {
		panic(err) // this should never happen since we verify there are more leaves before calling calcRoot
	}
	var leftChild, rightChild, sibling Node
	for {
		if layer == stopAtLayer {
			break
		}
		if v.shouldCalcSubtree(idx, layer) {
			sibling = v.calcRoot(layer)
		} else {
			var err error
			sibling, err = v.proofNodes.next()
			if err == noMoreItems {
				break
			}
		}
		if leftSibling(idx, layer) {
			leftChild, rightChild = sibling, activeNode
		} else {
			leftChild, rightChild = activeNode, sibling
		}
		activeNode = GetSha256Parent(leftChild, rightChild)
		layer++
	}
	return activeNode
}

// leftSibling returns true if the sibling of the node at the current layer on the path to leaf with index idx is on the
// left.
func leftSibling(idx uint64, layer uint) bool {
	// Is the bit at layer+1 equal 1?
	return (idx>>layer)%2 == 1
}

// shouldCalcSubtree returns true if the paths to idx (current leaf) and the nextIdx (next one) diverge at the current
// layer, so the next sibling should be the root of the subtree to the right.
func (v *validator) shouldCalcSubtree(idx uint64, layer uint) bool {
	nextIdx, err := v.leaves.peek()
	if err == noMoreItems {
		return false
	}
	// When eliminating the `layer` most insignificant bits of the bitwise xor of the current and next leaf index we
	// expect to get 1 at the divergence point.
	return (idx^nextIdx)>>layer == 1
}

var noMoreItems = errors.New("no more items")

type proofIterator struct {
	nodes []Node
}

func (it *proofIterator) next() (Node, error) {
	if len(it.nodes) == 0 {
		return nil, noMoreItems
	}
	n := it.nodes[0]
	it.nodes = it.nodes[1:]
	return n, nil
}

type leafIterator struct {
	indices []uint64
	leaves  []Node
}

// leafIterator.next() returns the leaf index and value
func (it *leafIterator) next() (uint64, Node, error) {
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
func (it *leafIterator) peek() (uint64, error) {
	if len(it.indices) == 0 {
		return 0, noMoreItems
	}
	idx := it.indices[0]
	return idx, nil
}
