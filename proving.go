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
	var found bool

	// Verify we got the base layer.
	if _, found = readers[0]; !found {
		return nil, errors.New("reader for base layer must be included")
	}

	provenLeafIndexIt := &positionsIterator{s: provenLeafIndices}
	var currentPos, subtreeStart position
	var width uint64
	var currentVal []byte
	skipPositions := &positionsStack{}
	rootHeight := rootHeightFromWidth(readers[0].Width())

	var proof, additionalProof [][]byte
	for ; ; currentPos = currentPos.parent() { // Process proven leaves:

		// Get the leaf whose subtree we'll traverse.
		nextProvenLeaf, err := provenLeafIndexIt.peek()
		if err == noMoreItems {
			// If there are no more leaves to prove - we're done.
			break
		}

		// Get the reader for the leaf layer.
		reader := readers[0]

		// Get indices for the bottom left corner of the subtree and its root, as well as the bottom layer's width.
		subtreeStart, currentPos, width = subtreeDefinition(nextProvenLeaf, readers)

		// Prepare reader to read subtree leaves.
		err = reader.Seek(subtreeStart.index)
		if err != nil {
			return nil, errors.New("while preparing to traverse subtree: " + err.Error())
		}

		// Prepare list of leaves to prove in the subtree.
		leavesToProve := provenLeafIndexIt.batchPopRelativeIndices(subtreeStart.index, width)

		// Traverse the subtree and append the additional proof nodes to the existing proof.
		additionalProof, _, err = traverseSubtree(reader, width, hash, leavesToProve)
		if err != nil {
			return nil, errors.New("while traversing subtree: " + err.Error())
		}
		proof = append(proof, additionalProof...)

		for ; currentPos.height < rootHeight; currentPos = currentPos.parent() { // Traverse cache:

			// Check if we're revisiting a node. If we've descended into a subtree and just got back, we shouldn't add
			// the sibling to the proof and instead move on to the parent.
			found = skipPositions.PopIfEqual(currentPos)
			if found {
				continue
			}

			// If the current node sibling is an ancestor of the next proven leaf sibling we should process it's subtree
			// instead of adding it to the proof. When we reach it again we'll want to skip it.
			if p, err := provenLeafIndexIt.peek(); err == nil && currentPos.sibling().isAncestorOf(p) {
				skipPositions.Push(currentPos.sibling())
				break
			}

			// Add the current node sibling to the proof:

			// Get the cache reader for the current layer.
			reader, found = readers[currentPos.height]

			// If the cache wan't found, we calculate the minimal subtree that will get us the required node.
			if !found {
				// Find the next cached layer below the current one.
				for subtreeStart = currentPos.leftChild(); !found; subtreeStart = subtreeStart.leftChild() {
					reader, found = readers[subtreeStart.height]
				}

				// Prepare the reader for traversing the subtree.
				err := reader.Seek(subtreeStart.index)
				if err != nil {
					return nil, errors.New("while seeking in cache: " + err.Error() + subtreeStart.String())
				}

				// Traverse the subtree.
				width = 1 << (currentPos.height - subtreeStart.height)
				_, currentVal, err = traverseSubtree(reader, width, hash, nil)
				if err != nil {
					return nil, errors.New("while traversing subtree for root: " + err.Error())
				}

				// Append the root of the subtree to the proof and move to its parent.
				proof = append(proof, currentVal)
				continue
			}

			// Read the current node sibling and add it to the proof.
			err = reader.Seek(currentPos.sibling().index)
			if err != nil {
				return nil, errors.New("while seeking in cache: " + err.Error() + currentPos.sibling().String())
			}
			currentVal, err = reader.ReadNext()
			if err != nil {
				return nil, errors.New("while reading from cache: " + err.Error())
			}
			proof = append(proof, currentVal)
		}
	}

	return proof, nil
}

// subtreeDefinition returns the definition (firstLeaf and root positions, width) for the minimal subtree whose
// base layer includes p and where the root is on a cached layer.
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

func (it *positionsIterator) next() (position, error) {
	if len(it.s) == 0 {
		return position{}, noMoreItems
	}
	index := it.s[0]
	it.s = it.s[1:]
	return position{index: index}, nil
}

func (it *positionsIterator) peek() (position, error) {
	if len(it.s) == 0 {
		return position{}, noMoreItems
	}
	index := it.s[0]
	return position{index: index}, nil
}

// batchPopRelativeIndices returns the indices, relative to the start index, of all positions starting at startIndex and
// spanning the requested width.
func (it *positionsIterator) batchPopRelativeIndices(startIndex, width uint64) []uint64 {
	var relativeIndices []uint64
	for len(it.s) > 0 && it.s[0] < startIndex+width {
		// By subtracting startIndex we get the relative index.
		relativeIndices = append(relativeIndices, it.s[0]-startIndex)
		it.s = it.s[1:]
	}
	return relativeIndices
}
