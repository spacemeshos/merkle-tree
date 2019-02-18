package merkle

import (
	"errors"
	"github.com/spacemeshos/sha256-simd"
)

type Tree interface {
	// AddLeaf updates the state of the tree with another leaf.
	AddLeaf(leaf Node)
	// Root returns the root of the tree or an error if the number of leaves added is not a power of 2.
	Root() (Node, error)
	// Proof returns a partial tree proving the membership of leaves that were passed in leavesToProve when the tree was
	// initialized or an error if the number of leaves added is not a power of 2. For a single proved leaf this is a
	// standard merkle proof (one sibling per layer of the tree from the leaves to the root, excluding the proved leaf
	// and root).
	Proof() ([]Node, error)
}

type incrementalTree struct {
	pendingLeftSiblings []Node
	currentLeaf         uint64
	leavesToProve       []uint64
	proof               []Node
}

// NewTree creates an empty tree structure that leaves can be added to. When all leaves have been added the root can be
// queried.
func NewTree() Tree {
	return &incrementalTree{
		pendingLeftSiblings: make([]Node, 0),
		currentLeaf:         0,
	}
}

// NewTree creates an empty tree structure that leaves can be added to. While the tree is constructed a single proof is
// generated that proves membership of all leaves included in leavesToProve. When all leaves have been added the root
// and proof can be queried.
func NewProvingTree(leavesToProve []uint64) Tree {
	return &incrementalTree{
		pendingLeftSiblings: make([]Node, 0),
		currentLeaf:         0,
		leavesToProve:       leavesToProve,
		proof:               make([]Node, 0),
	}
}

func (t *incrementalTree) AddLeaf(leaf Node) {
	activeNode := leaf
	for layer := 0; true; layer++ {
		// If pendingLeftSiblings is shorter than the current layer - extend it.
		if len(t.pendingLeftSiblings) == layer {
			t.pendingLeftSiblings = append(t.pendingLeftSiblings, nil)
		}
		// If we don't have a node waiting in pendingLeftSiblings, add the active node and break the loop.
		if t.pendingLeftSiblings[layer] == nil {
			t.pendingLeftSiblings[layer] = activeNode
			break
		}
		// If the active node should be in the proof - store it.
		t.addToProofIfNeeded(uint(layer), t.pendingLeftSiblings[layer], activeNode)
		// Since we found the active node's left sibling in pendingLeftSiblings we can calculate the parent, make it the
		// active node and move up a layer.
		activeNode = getParent(t.pendingLeftSiblings[layer], activeNode)
		// After using the left sibling we clear it from pendingLeftSiblings.
		t.pendingLeftSiblings[layer] = nil
	}
	t.currentLeaf++
}

func (t *incrementalTree) addToProofIfNeeded(currentLayer uint, leftChild, rightChild Node) {
	if len(t.leavesToProve) == 0 {
		// No proof was requested.
		return
	}
	// Calculate the paths to the parent and each child
	parentPath, leftChildPath, rightChildPath := getPaths(t.currentLeaf, currentLayer)
	if t.isNodeInProvedPath(parentPath, currentLayer+1) {
		// We need to be able to calculate the parent.
		// If the left child isn't in the proved path - we need to include it in the proof.
		if !t.isNodeInProvedPath(leftChildPath, currentLayer) {
			t.proof = append(t.proof, leftChild)
		}
		// If the right child isn't in the proved path - we need to include it in the proof.
		if !t.isNodeInProvedPath(rightChildPath, currentLayer) {
			t.proof = append(t.proof, rightChild)
		}
		// It's possible that both children are in the proved path and then we include none of them (and calculate them
		// instead).
	}
}

// getPaths uses the currentLeaf and layer to return the path from the root of the tree to the current node being added
// (parent) and each of its children, as a number: each binary digit represents a left (0) or right (1) turn.
func getPaths(currentLeaf uint64, layer uint) (parentPath, leftChildPath, rightChildPath uint64) {
	// This eliminates the layer+1 most insignificant digits, which represent the path from the current layer to the
	// bottom of the tree (the leaves).
	parentPath = currentLeaf / (1 << (layer + 1))
	// We then add a step in the path for the children with 0 for the left child and 1 for the right child.
	return parentPath, parentPath << 1, parentPath<<1 + 1
}

// getParent calculates the sha256 sum of child nodes to return their parent.
func getParent(leftChild, rightChild Node) Node {
	res := sha256.Sum256(append(leftChild, rightChild...))
	return res[:]
}

func (t *incrementalTree) Root() (Node, error) {
	for i, n := range t.pendingLeftSiblings {
		if i == len(t.pendingLeftSiblings)-1 {
			// We're at the end of the list and didn't encounter a pending left sibling - so this is the root.
			return n, nil
		}
		if n != nil {
			// If we found a non-nil sibling before the top of the list it means the leaf count isn't a power of 2.
			return nil, errors.New("number of leaves must be a power of 2")
		}
	}
	panic("we broke the laws of the universe!")
}

func (t *incrementalTree) Proof() ([]Node, error) {
	for i, n := range t.pendingLeftSiblings {
		if i == len(t.pendingLeftSiblings)-1 {
			// We're at the end of the list and didn't encounter a pending left sibling - so the proof is complete.
			return t.proof, nil
		}
		if n != nil {
			// If we found a non-nil sibling before the top of the list it means the leaf count isn't a power of 2.
			return nil, errors.New("number of leaves must be a power of 2")
		}
	}
	panic("we broke the laws of the universe!")
}

func (t *incrementalTree) isNodeInProvedPath(path uint64, layer uint) bool {
	// When we divide a leaf index by this divisor we get the path towards the leaf from the root to the current layer.
	var divisor uint64 = 1 << layer
	for _, leafToProve := range t.leavesToProve {
		if leafToProve/divisor == path {
			return true
		}
	}
	return false
}
