package merkle

import (
	"errors"
	"github.com/spacemeshos/sha256-simd"
	"io"
)

var ErrorIncompleteTree = errors.New("number of leaves must be a power of 2")

type node struct {
	value        []byte
	onProvenPath bool // Is this node an ancestor of a leaf whose membership in the tree is being proven?
}

type layer struct {
	height  uint
	parking node // This is where we park a node until its sibling is processed and we can calculate their parent.
	next    *layer
	cache   io.Writer
}

func (l *layer) ensureNextLayerExists(cache map[uint]io.Writer) {
	if l.next == nil {
		l.next = newLayer(l.height+1, cache[(l.height + 1)])
	}
}

func newLayer(height uint, cache io.Writer) *layer {
	return &layer{height: height, cache: cache}
}

type sparseBoolStack struct {
	sortedTrueIndices []uint64
	currentIndex      uint64
}

func (s *sparseBoolStack) pop() bool {
	if len(s.sortedTrueIndices) == 0 {
		return false
	}
	ret := s.currentIndex == s.sortedTrueIndices[0]
	if ret {
		s.sortedTrueIndices = s.sortedTrueIndices[1:]
	}
	s.currentIndex++
	return ret
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
	leavesToProve *sparseBoolStack
	cache         map[uint]io.Writer
}

func (t *Tree) calcParent(lChild, rChild node) node {
	return node{
		value:        t.hash(lChild.value, rChild.value),
		onProvenPath: lChild.onProvenPath || rChild.onProvenPath,
	}
}

// AddLeaf incorporates a new leaf to the state of the tree. It updates the state required to eventually determine the
// root of the tree and also updates the proof, if applicable.
func (t *Tree) AddLeaf(value []byte) error {
	err := t.addNode(node{
		value:        value,
		onProvenPath: t.leavesToProve.pop(),
	})
	return err
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

func (t *Tree) Proof() ([][]byte, error) {
	// We call t.Root() to traverse the layers and ensure the tree is full.
	if _, err := t.Root(); err != nil {
		return nil, err
	}
	return t.proof, nil
}

func (t *Tree) addNode(n node) error {
	var parent, lChild, rChild node
	l := t.baseLayer
	var lastCachingError error
	for {
		if l.cache != nil {
			_, err := l.cache.Write(n.value)
			if err != nil {
				lastCachingError = errors.New("error while caching: " + err.Error())
			}
		}
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
			l.ensureNextLayerExists(t.cache)
			l = l.next
		}
	}
	return lastCachingError
}

func NewTree(hash func(lChild, rChild []byte) []byte) *Tree {
	return NewCachingTree(hash, make(map[uint]io.Writer))
}

func NewProvingTree(hash func(lChild, rChild []byte) []byte, sortedLeavesToProve []uint64) *Tree {
	t := NewTree(hash)
	t.leavesToProve.sortedTrueIndices = sortedLeavesToProve
	return t
}

func NewCachingTree(hash func(lChild, rChild []byte) []byte, cache map[uint]io.Writer) *Tree {
	t := &Tree{
		hash: hash,
		baseLayer: newLayer(0, cache[0]),
		leavesToProve: &sparseBoolStack{},
	}
	t.cache = cache
	return t
}

func GetSha256Parent(lChild, rChild []byte) []byte {
	res := sha256.Sum256(append(lChild, rChild...))
	return res[:]
}
