package merkle

import (
	"errors"
	"math"
)

const NodeSize = 32

type NodeReader interface {
	Seek(index uint64) error
	ReadNext() ([]byte, error)
	Width() uint64
}

func GenerateProof(
	provenLeafIndices []uint64,
	readers map[uint]NodeReader,
	hash HashFunc,
) ([][]byte, error) {

	var proof [][]byte

	provenLeafIndexIt := &positionsIterator{s: provenLeafIndices}
	skipPositions := &positionsStack{}
	rootHeight := rootHeightFromWidth(readers[0].Width())

	cache, err := NewTreeCache(readers, hash)
	if err != nil {
		return nil, err
	}

	for { // Process proven leaves:

		// Get the leaf whose subtree we'll traverse.
		nextProvenLeafPos, found := provenLeafIndexIt.peek()
		if !found {
			// If there are no more leaves to prove - we're done.
			break
		}

		additionalProof, currentPos, err := cache.calcProofForNextLeaf(nextProvenLeafPos, provenLeafIndexIt)
		if err != nil {
			return nil, err
		}
		proof = append(proof, additionalProof...)

		for ; currentPos.height < rootHeight; currentPos = currentPos.parent() { // Traverse cache:

			// Check if we're revisiting a node. If we've descended into a subtree and just got back, we shouldn't add
			// the sibling to the proof and instead move on to the parent.
			found := skipPositions.PopIfEqual(currentPos)
			if found {
				continue
			}

			// If the current node sibling is an ancestor of the next proven leaf sibling we should process it's subtree
			// instead of adding it to the proof. When we reach it again we'll want to skip it.
			if p, found := provenLeafIndexIt.peek(); found && currentPos.sibling().isAncestorOf(p) {
				skipPositions.Push(currentPos.sibling())
				break
			}

			currentVal, err := cache.GetNode(currentPos.sibling())
			if err != nil {
				return nil, err
			}

			proof = append(proof, currentVal)
		}
	}

	return proof, nil
}

type TreeCache struct {
	readers map[uint]NodeReader
	hash    HashFunc
}

// GetNode reads the node at the requested position from the cache or calculates it if not available.
func (c *TreeCache) GetNode(nodePos position) ([]byte, error) {
	// Get the cache reader for the requested node's layer.
	reader, found := c.readers[nodePos.height]

	// If the cache wan't found, we calculate the minimal subtree that will get us the required node.
	if !found {
		return c.calcNode(nodePos)
	}

	err := reader.Seek(nodePos.index)
	if err != nil {
		return nil, errors.New("while seeking in cache: " + err.Error() + nodePos.String())
	}
	currentVal, err := reader.ReadNext()
	if err != nil {
		return nil, errors.New("while reading from cache: " + err.Error())
	}
	return currentVal, nil
}

func (c *TreeCache) calcNode(nodePos position) ([]byte, error) {
	var subtreeStart position
	var found bool
	var reader NodeReader

	// Find the next cached layer below the current one.
	for subtreeStart = nodePos.leftChild(); !found; subtreeStart = subtreeStart.leftChild() {
		reader, found = c.readers[subtreeStart.height]
	}

	// Prepare the reader for traversing the subtree.
	err := reader.Seek(subtreeStart.index)
	if err != nil {
		return nil, errors.New("while seeking in cache: " + err.Error() + subtreeStart.String())
	}

	// Traverse the subtree.
	width := uint64(1) << (nodePos.height - subtreeStart.height)
	_, currentVal, err := traverseSubtree(reader, width, c.hash, nil)
	if err != nil {
		return nil, errors.New("while traversing subtree for root: " + err.Error())
	}
	return currentVal, nil
}

func (c *TreeCache) calcProofForNextLeaf(nextProvenLeafPos position, provenLeafIndexIt *positionsIterator) ([][]byte,
	position, error) {

	// Get the reader for the leaf layer.
	reader := c.readers[0]

	// Get indices for the bottom left corner of the subtree and its root, as well as the bottom layer's width.
	subtreeStart, currentPos, width := subtreeDefinition(nextProvenLeafPos, c.readers)

	// Prepare reader to read subtree leaves.
	err := reader.Seek(subtreeStart.index)
	if err != nil {
		return nil, position{}, errors.New("while preparing to traverse subtree: " + err.Error())
	}

	// Prepare list of leaves to prove in the subtree.
	leavesToProve := provenLeafIndexIt.batchPop(subtreeStart.index + width)

	// By subtracting subtreeStart.index we get the index relative to the subtree.
	for i, leafIndex := range leavesToProve {
		leavesToProve[i] = leafIndex - subtreeStart.index
	}

	// Traverse the subtree and append the additional proof nodes to the existing proof.
	additionalProof, _, err := traverseSubtree(reader, width, c.hash, leavesToProve)
	if err != nil {
		return nil, position{}, errors.New("while traversing subtree: " + err.Error())
	}

	return additionalProof, currentPos, err
}

func NewTreeCache(readers map[uint]NodeReader, hash HashFunc) (*TreeCache, error) {
	// Verify we got the base layer.
	if _, found := readers[0]; !found {
		return nil, errors.New("reader for base layer must be included")
	}

	return &TreeCache{
		readers: readers,
		hash:    hash,
	}, nil
}

// subtreeDefinition returns the definition (firstLeaf and root positions, width) for the minimal subtree whose
// base layer includes p and where the root is on a cached layer. If no cached layer exists above the base layer, the
// subtree will reach the root of the original tree.
func subtreeDefinition(p position, readers map[uint]NodeReader) (firstLeaf, root position, width uint64) {
	// maxRootHeight represents the max height of the tree, based on the width of base layer. This is used to prevent an
	// infinite loop.
	maxRootHeight := rootHeightFromWidth(readers[p.height].Width())
	for root = p.parent(); root.height < maxRootHeight; root = root.parent() {
		if _, found := readers[root.height]; found {
			break
		}
	}
	subtreeHeight := root.height - p.height
	firstLeaf = position{
		index:  root.index << subtreeHeight,
		height: p.height,
	}
	return firstLeaf, root, 1 << subtreeHeight
}

func traverseSubtree(leafReader NodeReader, width uint64, hash HashFunc,
	leavesToProve []uint64) ([][]byte, []byte, error) {

	t := NewProvingTree(hash, leavesToProve)
	for i := uint64(0); i < width; i++ {
		leaf, err := leafReader.ReadNext()
		if err != nil {
			return nil, nil, errors.New("while reading a leaf: " + err.Error())
		}
		err = t.AddLeaf(leaf)
		if err != nil {
			return nil, nil, errors.New("while adding a leaf: " + err.Error())
		}
	}
	proof, err := t.Proof()
	if err != nil {
		return nil, nil, errors.New("while fetching the proof: " + err.Error())
	}
	root, err := t.Root()
	if err != nil {
		return nil, nil, errors.New("while fetching the root: " + err.Error())
	}
	return proof, root, nil
}

func rootHeightFromWidth(width uint64) uint {
	return uint(math.Ceil(math.Log2(float64(width))))
}

type positionsStack struct {
	positions []position
}

func (s *positionsStack) Push(v position) {
	s.positions = append(s.positions, v)
}

// Check the top of the stack for equality and pop the element if it's equal.
func (s *positionsStack) PopIfEqual(p position) bool {
	l := len(s.positions)
	if l == 0 {
		return false
	}
	if s.positions[l-1] == p {
		s.positions = s.positions[:l-1]
		return true
	}
	return false
}

type positionsIterator struct {
	s []uint64
}

func (it *positionsIterator) next() (pos position, found bool) {
	if len(it.s) == 0 {
		return position{}, false
	}
	index := it.s[0]
	it.s = it.s[1:]
	return position{index: index}, true
}

func (it *positionsIterator) peek() (pos position, found bool) {
	if len(it.s) == 0 {
		return position{}, false
	}
	index := it.s[0]
	return position{index: index}, true
}

// batchPop returns the indices of all positions up to endIndex.
func (it *positionsIterator) batchPop(endIndex uint64) []uint64 {
	var res []uint64
	for len(it.s) > 0 && it.s[0] < endIndex {
		res = append(res, it.s[0])
		it.s = it.s[1:]
	}
	return res
}
