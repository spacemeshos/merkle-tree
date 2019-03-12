package merkle

import (
	"errors"
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

		// Get indices for the bottom left corner of the subtree and its root, as well as the bottom layer's width.
		currentPos, subtreeStart, width := cache.subtreeDefinition(nextProvenLeafPos)

		// Prepare list of leaves to prove in the subtree.
		leavesToProve := provenLeafIndexIt.batchPop(subtreeStart.index + width)

		additionalProof, err := calcSubtreeProof(cache, hash, leavesToProve, subtreeStart, width)
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

func calcSubtreeProof(cache *TreeCache, hash HashFunc, leavesToProve []uint64, subtreeStart position, width uint64) (
	[][]byte, error) {

	// By subtracting subtreeStart.index we get the index relative to the subtree.
	relativeLeavesToProve := make([]uint64, len(leavesToProve))
	for i, leafIndex := range leavesToProve {
		relativeLeavesToProve[i] = leafIndex - subtreeStart.index
	}

	// Prepare leaf reader to read subtree leaves.
	reader := cache.LeafReader()
	err := reader.Seek(subtreeStart.index)
	if err != nil {
		return nil, errors.New("while preparing to traverse subtree: " + err.Error())
	}

	additionalProof, _, err := traverseSubtree(reader, width, hash, relativeLeavesToProve)
	if err != nil {
		return nil, errors.New("while traversing subtree: " + err.Error())
	}

	return additionalProof, err
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
