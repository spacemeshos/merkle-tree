package merkle

import (
	"errors"
	"github.com/spacemeshos/sha256-simd"
)

// Tree calculates a merkle tree root. It can optionally calculate a proof, or partial tree, for leaves defined in
// advance. Leaves are appended to the tree incrementally. It uses O(log(n)) memory to calculate the root and
// O(k*log(n)) (k being the number of leaves to prove) memory to calculate proofs.
//
// Tree is NOT thread safe.
//
// It has the following methods:
//
// 	AddLeaf(leaf Node)
// AddLeaf updates the state of the tree with another leaf.
//
//	Root() (Node, error)
// Root returns the root of the tree or an error if the number of leaves added is not a power of 2.
//
//	Proof() ([]Node, error)
// Proof returns a partial tree proving the membership of leaves that were passed in leavesToProve when the tree was
// initialized or an error if the number of leaves added is not a power of 2. For a single proved leaf this is a
// standard merkle proof (one sibling per layer of the tree from the leaves to the root, excluding the proved leaf
// and root).
type Tree struct {
	pendingLeftSiblings []Node
	currentLeaf         uint64
	leavesToProve       []uint64
	proof               []Node
}

// NewTree creates an empty tree structure that leaves can be added to. When all leaves have been added the root can be
// queried.
func NewTree() Tree {
	return Tree{
		pendingLeftSiblings: make([]Node, 0),
		currentLeaf:         0,
	}
}

// NewProvingTree creates an empty tree structure that leaves can be added to. While the tree is constructed a single
// proof is generated that proves membership of all leaves included in leavesToProve. When all leaves have been added
// the root and proof can be queried.
func NewProvingTree(leavesToProve []uint64) Tree {
	return Tree{
		pendingLeftSiblings: make([]Node, 0),
		currentLeaf:         0,
		leavesToProve:       leavesToProve,
		proof:               make([]Node, 0),
	}
}

// AddLeaf incorporates a new leaf to the state of the tree. It updates the state required to eventually determine the
// root of the tree and also updates the proof, if applicable.
func (t *Tree) AddLeaf(leaf Node) {
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

// addToProofIfNeeded considers the currently-being-processed nodes for inclusion in the proof. It uses the tree's
// currentLeaf and the received currentLayer to determine if any of the two nodes currently in memory needs to be needs
// to be appended to the tree's proof slice.
func (t *Tree) addToProofIfNeeded(currentLayer uint, leftChild, rightChild Node) {
	if len(t.leavesToProve) == 0 {
		// No proof was requested.
		return
	}
	// Calculate the paths to the parent and each child
	parentPath, leftChildPath, rightChildPath := getPathsToNodeAndItsChildren(t.currentLeaf, currentLayer+1)
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

// getPathsToNodeAndItsChildren returns the path from the root of the tree to leaf up to the requested layer (node). It
// also returns the path to each of the node's children.
// Paths are represented as numbers: each binary digit represents a left (0) or right (1) turn.
func getPathsToNodeAndItsChildren(leaf uint64, layer uint) (nodePath, leftChildPath, rightChildPath uint64) {
	// This eliminates the layer most insignificant digits, which represent the path from the requested layer to the
	// bottom of the tree (the leaves).
	nodePath = leaf >> layer
	// We then add a step in the path for the children with 0 for the left child and 1 for the right child.
	return nodePath, nodePath << 1, nodePath<<1 + 1
}

// getParent calculates the sha256 sum of child nodes to return their parent.
func getParent(leftChild, rightChild Node) Node {
	res := sha256.Sum256(append(leftChild, rightChild...))
	return res[:]
}

func (t *Tree) Root() (Node, error) {
	if t.isFull() {
		return t.pendingLeftSiblings[len(t.pendingLeftSiblings)-1], nil
	} else {
		return nil, errors.New("number of leaves must be a power of 2")
	}
}

func (t *Tree) Proof() ([]Node, error) {
	if t.isFull() {
		return t.proof, nil
	} else {
		return nil, errors.New("number of leaves must be a power of 2")
	}
}

func (t *Tree) isFull() bool {
	for i, n := range t.pendingLeftSiblings {
		if i == len(t.pendingLeftSiblings)-1 {
			// We're at the end of the list and didn't encounter a pending left sibling - so the tree is full.
			return true
		}
		if n != nil {
			// If we found a non-nil sibling before the top of the list it means the leaf count isn't a power of 2.
			return false
		}
	}
	// Tree is empty.
	return false
}

func (t *Tree) isNodeInProvedPath(path uint64, layer uint) bool {
	for _, leafToProve := range t.leavesToProve {
		// When we shift a leaf index right by the layer we get the path towards the leaf from the root to the current
		// layer.
		if leafToProve>>layer == path {
			return true
		}
	}
	return false
}
