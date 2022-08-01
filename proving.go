package merkle

import (
	"errors"
	"fmt"
	"io"
)

var ErrMissingValueAtBaseLayer = errors.New("reader for base layer must be included")

func GenerateProof(
	provenLeafIndices map[uint64]bool,
	treeCache CacheReader,
) (sortedProvenLeafIndices []uint64, provenLeaves, proofNodes [][]byte, err error) {

	provenLeafIndexIt := NewPositionsIterator(provenLeafIndices)
	skipPositions := &positionsStack{}
	width, err := treeCache.GetLayerReader(0).Width()
	if err != nil {
		return nil, nil, nil, err
	}
	rootHeight := RootHeightFromWidth(width)

	for { // Process proven leaves:

		// Get the leaf whose subtree we'll traverse.
		nextProvenLeafPos, found := provenLeafIndexIt.peek()
		if !found {
			// If there are no more leaves to prove - we're done.
			break
		}

		// Get indices for the bottom left corner of the subtree and its root, as well as the bottom layer's width.
		currentPos, subtreeStart, width, err := subtreeDefinition(treeCache, nextProvenLeafPos)
		if err != nil {
			return nil, nil, nil, err
		}

		// Prepare list of leaves to prove in the subtree.
		leavesToProve := provenLeafIndexIt.batchPop(subtreeStart.Index + width)

		additionalProof, additionalLeaves, err := calcSubtreeProof(treeCache, leavesToProve, subtreeStart, width)
		if err != nil {
			return nil, nil, nil, err
		}
		proofNodes = append(proofNodes, additionalProof...)
		provenLeaves = append(provenLeaves, additionalLeaves...)

		for ; currentPos.Height < rootHeight; currentPos = currentPos.parent() { // Traverse treeCache:

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
			currentVal, err := GetNode(treeCache, currentPos.sibling())
			if err != nil {
				return nil, nil, nil, err
			}
			proofNodes = append(proofNodes, currentVal)
		}
	}

	return Set(provenLeafIndices).AsSortedSlice(), provenLeaves, proofNodes, nil
}

func calcSubtreeProof(c CacheReader, leavesToProve Set, subtreeStart Position, width uint64) (
	additionalProof, additionalLeaves [][]byte, err error) {

	// By subtracting subtreeStart.index we get the index relative to the subtree.
	relativeLeavesToProve := make(Set)
	for leafIndex, prove := range leavesToProve {
		relativeLeavesToProve[leafIndex-subtreeStart.Index] = prove
	}

	// Prepare leaf reader to read subtree leaves.
	reader := c.GetLayerReader(0)
	err = reader.Seek(subtreeStart.Index)
	if err != nil {
		return nil, nil, errors.New("while preparing to traverse subtree: " + err.Error())
	}

	_, additionalProof, additionalLeaves, err = traverseSubtree(reader, width, c.GetHashFunc(), relativeLeavesToProve, nil)
	if err != nil {
		return nil, nil, errors.New("while traversing subtree: " + err.Error())
	}

	return additionalProof, additionalLeaves, err
}

func traverseSubtree(leafReader LayerReader, width uint64, hash HashFunc, leavesToProve Set,
	externalPadding []byte) (root []byte, proof, provenLeaves [][]byte, err error) {

	shouldUseExternalPadding := externalPadding != nil
	t, err := NewTreeBuilder().
		WithHashFunc(hash).
		WithLeavesToProve(leavesToProve).
		WithMinHeight(RootHeightFromWidth(width)). // This ensures the correct size tree, even if padding is needed.
		Build()
	if err != nil {
		return nil, nil, nil, errors.New("while building a tree: " + err.Error())
	}
	for i := uint64(0); i < width; i++ {
		leaf, err := leafReader.ReadNext()
		if err == io.EOF {
			// Add external padding if provided.
			if !shouldUseExternalPadding {
				break
			}
			leaf = externalPadding
			shouldUseExternalPadding = false
		} else if err != nil {
			return nil, nil, nil, errors.New("while reading a leaf: " + err.Error())
		}
		err = t.AddLeaf(leaf)
		if err != nil {
			return nil, nil, nil, errors.New("while adding a leaf: " + err.Error())
		}
		if leavesToProve[i] {
			provenLeaves = append(provenLeaves, leaf)
		}
	}
	root, proof = t.RootAndProof()
	return root, proof, provenLeaves, nil
}

// GetNode reads the node at the requested Position from the cache or calculates it if not available.
func GetNode(c CacheReader, nodePos Position) ([]byte, error) {
	// Get the cache reader for the requested node's layer.
	reader := c.GetLayerReader(nodePos.Height)
	// If the cache wasn't found, we calculate the minimal subtree that will get us the required node.
	if reader == nil {
		return calcNode(c, nodePos)
	}

	err := reader.Seek(nodePos.Index)
	if err == io.EOF {
		return calcNode(c, nodePos)
	}
	if err != nil {
		return nil, errors.New("while seeking to Position " + nodePos.String() + " in cache: " + err.Error())
	}
	currentVal, err := reader.ReadNext()
	if err != nil {
		return nil, errors.New("while reading from cache: " + err.Error())
	}
	return currentVal, nil
}

func calcNode(c CacheReader, nodePos Position) ([]byte, error) {
	if nodePos.Height == 0 {
		return nil, ErrMissingValueAtBaseLayer
	}
	// Find the next cached layer below the current one.
	var subtreeStart = nodePos
	var reader LayerReader
	for {
		subtreeStart = subtreeStart.leftChild()
		fmt.Println(subtreeStart.Height)
		reader = c.GetLayerReader(subtreeStart.Height)

		err := reader.Seek(subtreeStart.Index)
		if err == nil {
			break
		}
		if err != nil && err != io.EOF {
			return nil, errors.New("while seeking to Position " + subtreeStart.String() + " in cache: " + err.Error())
		}
		if subtreeStart.Height == 0 {
			return PaddingValue.value, nil
		}
	}

	var paddingValue []byte
	width := uint64(1) << (nodePos.Height - subtreeStart.Height)
	readerWidth, err := reader.Width()
	if err != nil {
		return nil, fmt.Errorf("while getting reader width: %v", err)
	}
	if readerWidth < subtreeStart.Index+width {
		paddingPos := Position{
			Index:  readerWidth,
			Height: subtreeStart.Height,
		}
		paddingValue, err = calcNode(c, paddingPos)
		if err == ErrMissingValueAtBaseLayer {
			paddingValue = PaddingValue.value
		} else if err != nil {
			return nil, errors.New("while calculating ephemeral node at Position " + paddingPos.String() + ": " + err.Error())
		}
	}

	// Traverse the subtree.
	currentVal, _, _, err := traverseSubtree(reader, width, c.GetHashFunc(), nil, paddingValue)
	if err != nil {
		return nil, errors.New("while traversing subtree for root: " + err.Error())
	}
	return currentVal, nil
}

// subtreeDefinition returns the definition (firstLeaf and root positions, width) for the minimal subtree whose
// base layer includes p and where the root is on a cached layer. If no cached layer exists above the base layer, the
// subtree will reach the root of the original tree.
func subtreeDefinition(c CacheReader, p Position) (root, firstLeaf Position, width uint64, err error) {
	// maxRootHeight represents the max height of the tree, based on the width of base layer. This is used to prevent an
	// infinite loop.
	width, err = c.GetLayerReader(p.Height).Width()
	if err != nil {
		return Position{}, Position{}, 0, err
	}
	maxRootHeight := RootHeightFromWidth(width)
	root = p
	if !(p.Height == 0 && width == 1) {
		// TODO(dshulyak) doublecheck with @noam if there is more generic fix
		// failing test case go test ./ -run=TestValidatePartialTreeProofs/N1/L0/Cache -v
		for root = p.parent(); root.Height < maxRootHeight; root = root.parent() {
			if layer := c.GetLayerReader(root.Height); layer != nil {
				break
			}
		}
	}
	subtreeHeight := root.Height - p.Height
	firstLeaf = Position{
		Index:  root.Index << subtreeHeight,
		Height: p.Height,
	}
	return root, firstLeaf, 1 << subtreeHeight, err
}
