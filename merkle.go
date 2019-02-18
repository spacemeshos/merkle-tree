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
	path          []Node
	currentLeaf   uint64
	leavesToProve []uint64
	proof         []Node
}

// NewTree creates an empty tree structure that leaves can be added to. When all leaves have been added the root can be
// queried.
func NewTree() Tree {
	return &incrementalTree{
		path:        make([]Node, 0),
		currentLeaf: 0,
	}
}

// NewTree creates an empty tree structure that leaves can be added to. While the tree is constructed a single proof is
// generated that proves membership of all leaves included in leavesToProve. When all leaves have been added the root
// and proof can be queried.
func NewProvingTree(leavesToProve []uint64) Tree {
	return &incrementalTree{
		path:          make([]Node, 0),
		currentLeaf:   0,
		leavesToProve: leavesToProve,
		proof:         make([]Node, 0),
	}
}

func (t *incrementalTree) AddLeaf(leaf Node) {
	activeNode := leaf
	for i := 0; true; i++ {
		if len(t.path) == i {
			t.path = append(t.path, nil)
		}
		if t.path[i] == nil {
			t.path[i] = activeNode
			break
		}
		t.addToProofIfNeeded(uint(i), t.path[i], activeNode)
		activeNode = getParent(t.path[i], activeNode)
		t.path[i] = nil
	}
	t.currentLeaf++
}

func (t *incrementalTree) addToProofIfNeeded(currentLayer uint, leftChild, rightChild Node) {
	if len(t.leavesToProve) == 0 {
		return
	}
	parentPath, leftChildPath, rightChildPath := getPaths(t.currentLeaf, currentLayer)
	if t.isNodeInProvedPath(parentPath, currentLayer+1) {
		if !t.isNodeInProvedPath(leftChildPath, currentLayer) {
			t.proof = append(t.proof, leftChild)
		}
		if !t.isNodeInProvedPath(rightChildPath, currentLayer) {
			t.proof = append(t.proof, rightChild)
		}
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
	for i, n := range t.path {
		if i == len(t.path)-1 {
			return n, nil
		}
		if n != nil {
			return nil, errors.New("number of leaves must be a power of 2")
		}
	}
	panic("we broke the laws of the universe!")
}

func (t *incrementalTree) Proof() ([]Node, error) {
	for i, n := range t.path {
		if i == len(t.path)-1 {
			return t.proof, nil
		}
		if n != nil {
			return nil, errors.New("number of leaves must be a power of 2")
		}
	}
	panic("we broke the laws of the universe!")
}

func (t *incrementalTree) isNodeInProvedPath(path uint64, layer uint) bool {
	var divisor uint64 = 1 << layer
	for _, leafToProve := range t.leavesToProve {
		if leafToProve/divisor == path {
			return true
		}
	}
	return false
}
