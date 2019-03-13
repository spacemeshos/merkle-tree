package merkle

import (
	"errors"
	"github.com/spacemeshos/sha256-simd"
	"io"
	"sort"
)

// PaddingValue is used for padding unbalanced trees. This value should not be permitted at the leaf layer to
// distinguish padding from actual members of the tree.
var PaddingValue = node{
	value: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
}

// node is a node in the merkle tree.
type node struct {
	value        []byte
	onProvenPath bool // Whether this node is an ancestor of a leaf whose membership in the tree is being proven.
}

func (n node) IsEmpty() bool {
	return n.value == nil
}

// layer is a layer in the merkle tree.
type layer struct {
	height  uint
	parking node // This is where we park a node until its sibling is processed and we can calculate their parent.
	next    *layer
	cache   io.Writer
}

// ensureNextLayerExists creates the next layer if it doesn't exist.
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

func newSparseBoolStack(trueIndices []uint64) *sparseBoolStack {
	sorted := make([]uint64, len(trueIndices))
	copy(sorted, trueIndices)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	return &sparseBoolStack{sortedTrueIndices: sorted}
}

func (s *sparseBoolStack) Pop() bool {
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

type HashFunc func(lChild, rChild []byte) []byte

// Tree calculates a merkle tree root. It can optionally calculate a proof, or partial tree, for leaves defined in
// advance. Leaves are appended to the tree incrementally. It uses O(log(n)) memory to calculate the root and
// O(k*log(n)) (k being the number of leaves to prove) memory to calculate proofs.
//
// Tree is NOT thread safe.
type Tree struct {
	baseLayer     *layer // The leaf layer (0)
	hash          HashFunc
	proof         [][]byte
	leavesToProve *sparseBoolStack
	cache         map[uint]io.Writer
	minHeight     uint
}

// AddLeaf incorporates a new leaf to the state of the tree. It updates the state required to eventually determine the
// root of the tree and also updates the proof, if applicable.
func (t *Tree) AddLeaf(value []byte) error {
	n := node{
		value:        value,
		onProvenPath: t.leavesToProve.Pop(),
	}
	l := t.baseLayer
	var parent, lChild, rChild node
	var lastCachingError error

	// Loop through the layers, starting from the base layer.
	for {
		// Writing the node to its layer cache, if applicable.
		if l.cache != nil {
			_, err := l.cache.Write(n.value)
			if err != nil {
				lastCachingError = errors.New("error while caching: " + err.Error())
			}
		}

		// If no node is pending, then this node is a left sibling,
		// pending for its right sibling before its parent can be calculated.
		if l.parking.IsEmpty() {
			l.parking = n
			break
		} else {
			// This node is a right sibling.
			lChild, rChild = l.parking, n
			parent = t.calcParent(lChild, rChild)

			// A given node is required in the proof if and only if its parent is an ancestor
			// of a leaf whose membership in the tree is being proven, but the given node isn't.
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

func nextOrEmptyLayer(l *layer) *layer {
	if l.next != nil {
		return l.next
	}
	return &layer{height: l.height + 1}
}

// Root returns the root of the tree.
// If the tree is unbalanced (num. of leaves is not a power of 2) it will perform padding on-the-fly.
func (t *Tree) Root() []byte {
	root, _ := t.RootAndProof()
	return root
}

// Proof returns a partial tree proving the membership of leaves that were passed in leavesToProve when the tree was
// initialized. For a single proved leaf this is a standard merkle proof (one sibling per layer of the tree from the
// leaves to the root, excluding the proved leaf and root).
// If the tree is unbalanced (num. of leaves is not a power of 2) it will perform padding on-the-fly.
func (t *Tree) Proof() [][]byte {
	_, proof := t.RootAndProof()
	return proof
}

// RootAndProof returns the root of the tree and a partial tree proving the membership of leaves that were passed in
// leavesToProve when the tree was initialized. For a single proved leaf this is a standard merkle proof (one sibling
// per layer of the tree from the leaves to the root, excluding the proved leaf and root).
// If the tree is unbalanced (num. of leaves is not a power of 2) it will perform padding on-the-fly.
func (t *Tree) RootAndProof() ([]byte, [][]byte) {
	ephemeralProof := t.proof
	var ephemeralNode node
	l := t.baseLayer
	for height := uint(0); height < t.minHeight || l != nil; height++ {

		// If we've reached the last layer and the ephemeral node is still empty, the tree is balanced and the parked
		// node is its root.
		// In any other case (minHeight not reached, or the tree is unbalanced) we want to add padding at this point.
		reachedMinHeight := height >= t.minHeight
		onLastLayer := l != nil && l.next == nil
		parkingIsBalancedTreeRoot := reachedMinHeight && onLastLayer && ephemeralNode.IsEmpty()
		if parkingIsBalancedTreeRoot {
			return l.parking.value, ephemeralProof
		}

		var parking node
		if l != nil {
			parking = l.parking
		}
		parent, lChild, rChild := t.calcEphemeralParent(parking, ephemeralNode)

		// Consider adding children to the ephemeralProof.
		if parent.onProvenPath {
			if !lChild.onProvenPath {
				ephemeralProof = append(ephemeralProof, lChild.value)
			}
			if !rChild.onProvenPath {
				ephemeralProof = append(ephemeralProof, rChild.value)
			}
		}
		ephemeralNode = parent
		if l != nil {
			l = l.next
		}
	}
	return ephemeralNode.value, ephemeralProof
}

// calcEphemeralParent calculates the parent using the layer parking and ephemeralNode. When one of those is missing it
// uses PaddingValue to pad. It returns the actual nodes used along with the parent.
func (t *Tree) calcEphemeralParent(parking, ephemeralNode node) (parent, lChild, rChild node) {
	switch {
	case !parking.IsEmpty() && !ephemeralNode.IsEmpty():
		lChild, rChild = parking, ephemeralNode

	case !parking.IsEmpty() && ephemeralNode.IsEmpty():
		lChild, rChild = parking, PaddingValue

	case parking.IsEmpty() && !ephemeralNode.IsEmpty():
		lChild, rChild = ephemeralNode, PaddingValue

	default: // both are empty
		return ephemeralNode, ephemeralNode, ephemeralNode
	}
	return t.calcParent(lChild, rChild), lChild, rChild
}

// calcParent returns the parent node of two child nodes.
func (t *Tree) calcParent(lChild, rChild node) node {
	return node{
		value:        t.hash(lChild.value, rChild.value),
		onProvenPath: lChild.onProvenPath || rChild.onProvenPath,
	}
}

type TreeBuilder struct {
	hash           HashFunc
	leavesToProves []uint64
	cache          map[uint]io.Writer
	minHeight      uint
}

func NewTreeBuilder(hash HashFunc) TreeBuilder {
	return TreeBuilder{hash: hash}
}

func (tb TreeBuilder) Build() *Tree {
	if tb.cache == nil {
		tb.cache = make(map[uint]io.Writer)
	}
	return &Tree{
		baseLayer:     newLayer(0, tb.cache[0]),
		hash:          tb.hash,
		leavesToProve: newSparseBoolStack(tb.leavesToProves),
		cache:         tb.cache,
		minHeight:     tb.minHeight,
	}
}

func (tb TreeBuilder) WithLeavesToProve(leavesToProves []uint64) TreeBuilder {
	tb.leavesToProves = leavesToProves
	return tb
}

func (tb TreeBuilder) WithCache(cache map[uint]io.Writer) TreeBuilder {
	tb.cache = cache
	return tb
}

func (tb TreeBuilder) WithMinHeight(minHeight uint) TreeBuilder {
	tb.minHeight = minHeight
	return tb
}

func NewTree(hash HashFunc) *Tree {
	return NewTreeBuilder(hash).Build()
}

func NewProvingTree(hash HashFunc, leavesToProves []uint64) *Tree {
	return NewTreeBuilder(hash).WithLeavesToProve(leavesToProves).Build()
}

func NewCachingTree(hash HashFunc, cache map[uint]io.Writer) *Tree {
	return NewTreeBuilder(hash).WithCache(cache).Build()
}

func GetSha256Parent(lChild, rChild []byte) []byte {
	res := sha256.Sum256(append(lChild, rChild...))
	return res[:]
}
