package simpler

import (
	"errors"
	"github.com/spacemeshos/sha256-simd"
)

var ErrorIncompleteTree = errors.New("number of leaves must be a power of 2")

type node struct {
	value      []byte
	pathToRoot bool
}

type layer struct {
	parking node
	next    *layer
}

func (l *layer) ensureNextLayerExists() {
	if l.next == nil {
		l.next = &layer{}
	}
}

type Tree struct {
	baseLayer     *layer
	hash          func(leftChild, rightChild []byte) []byte
	proof         [][]byte
	leavesToProve map[uint64]struct{}
	currentIndex  uint64
}

func (t *Tree) calcParent(left, right node) node {
	return node{
		value:      t.hash(left.value, right.value),
		pathToRoot: left.pathToRoot || right.pathToRoot,
	}
}

func (t *Tree) AddLeaf(value []byte) {
	// TODO replace the is-in-proof mechanism?
	t.addNode(node{
		value:      value,
		pathToRoot: t.isLeafInProof(t.currentIndex),
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
	// TODO ensure that the tree is full
	return t.proof, nil
}

func (t *Tree) addNode(n node) {
	var parent node
	l := t.baseLayer
	for {
		if l.parking.value == nil {
			l.parking = n
			break
		} else {
			parent = t.calcParent(l.parking, n)
			if parent.pathToRoot {
				if !l.parking.pathToRoot {
					t.proof = append(t.proof, l.parking.value)
				}
				if !n.pathToRoot {
					t.proof = append(t.proof, n.value)
				}
			}
			l.ensureNextLayerExists()
			l.parking.value = nil
			n = parent
			l = l.next
		}
	}
}

func NewTree(hash func(leftChild, rightChild []byte) []byte) *Tree {
	return NewProvingTree(hash, nil)
}

func NewProvingTree(hash func(leftChild, rightChild []byte) []byte, leavesToProve []uint64) *Tree {
	t := &Tree{hash: hash, leavesToProve: make(map[uint64]struct{}), baseLayer: &layer{}}
	for _, l := range leavesToProve {
		t.leavesToProve[l] = struct{}{}
	}
	return t
}

func GetSha256Parent(leftChild, rightChild []byte) []byte {
	res := sha256.Sum256(append(leftChild, rightChild...))
	return res[:]
}
