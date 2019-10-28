package merkle

import (
	"errors"
	"github.com/spacemeshos/merkle-tree/shared"
	"github.com/spacemeshos/sha256-simd"
)

const NodeSize = shared.NodeSize

type (
	HashFunc        = shared.HashFunc
	LayerWriter     = shared.LayerWriter
	LayerReader     = shared.LayerReader
	LayerReadWriter = shared.LayerReadWriter
	CacheWriter     = shared.CacheWriter
	CacheReader     = shared.CacheReader
)

var RootHeightFromWidth = shared.RootHeightFromWidth

var EmptyNode node

// PaddingValue is used for padding unbalanced trees. This value should not be permitted at the leaf layer to
// distinguish padding from actual members of the tree.
var PaddingValue = node{
	value:        make([]byte, NodeSize), // Zero filled.
	OnProvenPath: false,
}

// node is a node in the merkle tree.
type node struct {
	value        []byte
	OnProvenPath bool // Whether this node is an ancestor of a leaf whose membership in the tree is being proven.
}

func (n node) IsEmpty() bool {
	return n.value == nil
}

// layer is a layer in the merkle tree.
type layer struct {
	height  uint
	parking node // This is where we park a node until its sibling is processed and we can calculate their parent.
	next    *layer
	cache   LayerWriter
}

// ensureNextLayerExists creates the next layer if it doesn't exist.
func (l *layer) ensureNextLayerExists(cacheWriter shared.CacheWriter) error {
	if l.next == nil {
		writer, err := cacheWriter.GetLayerWriter(l.height + 1)
		if err != nil {
			return err
		}
		l.next = newLayer(l.height+1, writer)
	}
	return nil
}

func newLayer(height uint, cache LayerWriter) *layer {
	return &layer{height: height, cache: cache}
}

type sparseBoolStack struct {
	sortedTrueIndices []uint64
	currentIndex      uint64
}

func NewSparseBoolStack(trueIndices Set) *sparseBoolStack {
	sorted := trueIndices.AsSortedSlice()
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
	cacheWriter   CacheWriter
	minHeight     uint
}

// AddLeaf incorporates a new leaf to the state of the tree. It updates the state required to eventually determine the
// root of the tree and also updates the proof, if applicable.
func (t *Tree) AddLeaf(value []byte) error {
	n := node{
		value:        value,
		OnProvenPath: t.leavesToProve.Pop(),
	}
	l := t.baseLayer
	var parent, lChild, rChild node
	var lastCachingError error

	// Loop through the layers, starting from the base layer.
	for {
		// Writing the node to its layer cache, if applicable.
		if l.cache != nil {
			_, err := l.cache.Append(n.value)
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
			if parent.OnProvenPath {
				if !lChild.OnProvenPath {
					t.proof = append(t.proof, lChild.value)
				}
				if !rChild.OnProvenPath {
					t.proof = append(t.proof, rChild.value)
				}
			}

			l.parking.value = nil
			n = parent
			err := l.ensureNextLayerExists(t.cacheWriter)
			if err != nil {
				return err
			}
			l = l.next
		}
	}
	return lastCachingError
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

		// Consider adding children to the ephemeralProof. `onProvenPath` must be explicitly set -- an empty node has
		// the default value `false` and would never pass this point.
		if parent.OnProvenPath {
			if !lChild.OnProvenPath {
				ephemeralProof = append(ephemeralProof, lChild.value)
			}
			if !rChild.OnProvenPath {
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

func (t *Tree) GetParkedNodes() [][]byte {
	var ret [][]byte
	layer := t.baseLayer
	for {
		ret = append(ret, layer.parking.value)
		if layer.next == nil {
			break
		} else {
			layer = layer.next
		}
	}
	return ret
}

func (t *Tree) SetParkedNodes(nodes [][]byte) error {
	layer := t.baseLayer
	for i := 0; i < len(nodes); i++ {
		if nodes[i] != nil {
			layer.parking.value = nodes[i]
		}

		if i < len(nodes)-1 {
			err := layer.ensureNextLayerExists(t.cacheWriter)
			if err != nil {
				return err
			}
			layer = layer.next
		}
	}

	return nil
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
		return EmptyNode, EmptyNode, EmptyNode
	}
	return t.calcParent(lChild, rChild), lChild, rChild
}

// calcParent returns the parent node of two child nodes.
func (t *Tree) calcParent(lChild, rChild node) node {
	return node{
		value:        t.hash(lChild.value, rChild.value),
		OnProvenPath: lChild.OnProvenPath || rChild.OnProvenPath,
	}
}

func GetSha256Parent(lChild, rChild []byte) []byte {
	res := sha256.Sum256(append(lChild, rChild...))
	return res[:]
}
