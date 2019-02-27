package merkle

import (
	"errors"
	"github.com/spacemeshos/sha256-simd"
)

var ErrorIncompleteTree = errors.New("number of leaves must be a power of 2")

type node struct {
	value        []byte
	onProvenPath bool // Is this node an ancestor of a leaf whose membership in the tree is being proven?
}

type layer struct {
	parking node // This is where we park a node until its sibling is processed and we can calculate their parent.
	next    *layer
}

func (l *layer) ensureNextLayerExists() {
	if l.next == nil {
		l.next = &layer{}
	}
}

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
	baseLayer     *layer
	hash          func(lChild, rChild []byte) []byte
	proof         [][]byte
	leavesToProve map[uint64]struct{}
	currentIndex  uint64
}

func (t *Tree) calcParent(lChild, rChild node) node {
	return node{
		value:        t.hash(lChild.value, rChild.value),
		onProvenPath: lChild.onProvenPath || rChild.onProvenPath,
	}
}

// AddLeaf incorporates a new leaf to the state of the tree. It updates the state required to eventually determine the
// root of the tree and also updates the proof, if applicable.
func (t *Tree) AddLeaf(value []byte) {
	t.addNode(node{
		value:        value,
		onProvenPath: t.isLeafInProof(t.currentIndex),
	})
	t.currentIndex++
}

func (t *Tree) Root() ([]byte, error) {
	l := t.baseLayer
	for {
		if l.next == nil {
			return l.parking.value, nil
		}
		if l.parking.value != nil {
			return nil, ErrorIncompleteTree
		}
		l = l.next
	}
}

func (t *Tree) isLeafInProof(index uint64) bool {
	_, found := t.leavesToProve[index]
	return found
}

func (t *Tree) Proof() ([][]byte, error) {
	// We call t.Root() to traverse the layers and ensure the tree is full.
	if _, err := t.Root(); err != nil {
		return nil, err
	}
	return t.proof, nil
}

func (t *Tree) addNode(n node) {
	var parent, lChild, rChild node
	l := t.baseLayer
	for {
		if l.parking.value == nil {
			l.parking = n
			break
		} else {
			lChild, rChild = l.parking, n
			parent = t.calcParent(lChild, rChild)
			// A given node is required in the proof iff its parent is an ancestor of a leaf whose membership in the
			// tree is being proven, but the given node isn't.
			if parent.onProvenPath {
				if !lChild.onProvenPath {
					t.proof = append(t.proof, lChild.value)
				}
				if !rChild.onProvenPath {
					t.proof = append(t.proof, rChild.value)
				}
			}
			l.parking.value = nil
			n = parent
			l.ensureNextLayerExists()
			l = l.next
		}
	}
}

func NewTree(hash func(lChild, rChild []byte) []byte) *Tree {
	return NewProvingTree(hash, nil)
}

func NewProvingTree(hash func(lChild, rChild []byte) []byte, leavesToProve []uint64) *Tree {
	t := &Tree{hash: hash, leavesToProve: make(map[uint64]struct{}), baseLayer: &layer{}}
	for _, l := range leavesToProve {
		t.leavesToProve[l] = struct{}{}
	}
	return t
}

func GetSha256Parent(lChild, rChild []byte) []byte {
	res := sha256.Sum256(append(lChild, rChild...))
	return res[:]
}
