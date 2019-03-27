package merkle

import (
	"errors"
	"github.com/spacemeshos/merkle-tree/cache"
	"io"
)

var ErrMissingValueAtBaseLayer = errors.New("reader for base layer must be included")

func GenerateProof(
	provenLeafIndices map[uint64]bool,
	treeCache *cache.Reader,
) (sortedProvenLeafIndices []uint64, provenLeaves, proofNodes [][]byte, err error) {

	provenLeafIndexIt := newPositionsIterator(provenLeafIndices)
	skipPositions := &positionsStack{}
	rootHeight := cache.RootHeightFromWidth(treeCache.GetLayerReader(0).Width())

	for { // Process proven leaves:

		// Get the leaf whose subtree we'll traverse.
		nextProvenLeafPos, found := provenLeafIndexIt.peek()
		if !found {
			// If there are no more leaves to prove - we're done.
			break
		}

		// Get indices for the bottom left corner of the subtree and its root, as well as the bottom layer's width.
		currentPos, subtreeStart, width := subtreeDefinition(treeCache, nextProvenLeafPos)

		// Prepare list of leaves to prove in the subtree.
		leavesToProve := provenLeafIndexIt.batchPop(subtreeStart.index + width)

		additionalProof, additionalLeaves, err := calcSubtreeProof(treeCache, leavesToProve, subtreeStart, width)
		if err != nil {
			return nil, nil, nil, err
		}
		proofNodes = append(proofNodes, additionalProof...)
		provenLeaves = append(provenLeaves, additionalLeaves...)

		for ; currentPos.height < rootHeight; currentPos = currentPos.parent() { // Traverse treeCache:

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

	return set(provenLeafIndices).asSortedSlice(), provenLeaves, proofNodes, nil
}

func calcSubtreeProof(c *cache.Reader, leavesToProve set, subtreeStart position, width uint64) (
	additionalProof, additionalLeaves [][]byte, err error) {

	// By subtracting subtreeStart.index we get the index relative to the subtree.
	relativeLeavesToProve := make(set)
	for leafIndex, prove := range leavesToProve {
		relativeLeavesToProve[leafIndex-subtreeStart.index] = prove
	}

	// Prepare leaf reader to read subtree leaves.
	reader := c.GetLayerReader(0)
	err = reader.Seek(subtreeStart.index)
	if err != nil {
		return nil, nil, errors.New("while preparing to traverse subtree: " + err.Error())
	}

	_, additionalProof, additionalLeaves, err = traverseSubtree(reader, width, c.GetHashFunc(), relativeLeavesToProve, nil)
	if err != nil {
		return nil, nil, errors.New("while traversing subtree: " + err.Error())
	}

	return additionalProof, additionalLeaves, err
}

func traverseSubtree(leafReader cache.LayerReader, width uint64, hash HashFunc, leavesToProve set,
	externalPadding []byte) (root []byte, proof, provenLeaves [][]byte, err error) {

	shouldUseExternalPadding := externalPadding != nil
	t := NewTreeBuilder().
		WithHashFunc(hash).
		WithLeavesToProve(leavesToProve).
		WithMinHeight(cache.RootHeightFromWidth(width)). // This ensures the correct size tree, even if padding is needed.
		Build()
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

// GetNode reads the node at the requested position from the cache or calculates it if not available.
func GetNode(c *cache.Reader, nodePos position) ([]byte, error) {
	// Get the cache reader for the requested node's layer.
	reader := c.GetLayerReader(nodePos.height)

	// If the cache wasn't found, we calculate the minimal subtree that will get us the required node.
	if reader == nil {
		return calcNode(c, nodePos)
	}

	err := reader.Seek(nodePos.index)
	if err == io.EOF {
		return calcNode(c, nodePos)
	}
	if err != nil {
		return nil, errors.New("while seeking to position " + nodePos.String() + " in cache: " + err.Error())
	}
	currentVal, err := reader.ReadNext()
	if err != nil {
		return nil, errors.New("while reading from cache: " + err.Error())
	}
	return currentVal, nil
}

func calcNode(c *cache.Reader, nodePos position) ([]byte, error) {
	var subtreeStart position
	var reader cache.LayerReader

	if nodePos.height == 0 {
		return nil, ErrMissingValueAtBaseLayer
	}

	// Find the next cached layer below the current one.
	for subtreeStart = nodePos; reader == nil; {
		subtreeStart = subtreeStart.leftChild()
		reader = c.GetLayerReader(subtreeStart.height)
	}

	// Prepare the reader for traversing the subtree.
	err := reader.Seek(subtreeStart.index)
	if err == io.EOF {
		return PaddingValue.value, nil
	}
	if err != nil {
		return nil, errors.New("while seeking to position " + subtreeStart.String() + " in cache: " + err.Error())
	}

	var paddingValue []byte
	width := uint64(1) << (nodePos.height - subtreeStart.height)
	if reader.Width() < subtreeStart.index+width {
		paddingPos := position{
			index:  reader.Width(),
			height: subtreeStart.height,
		}
		paddingValue, err = calcNode(c, paddingPos)
		if err == ErrMissingValueAtBaseLayer {
			paddingValue = PaddingValue.value
		} else if err != nil {
			return nil, errors.New("while calculating ephemeral node at position " + paddingPos.String() + ": " + err.Error())
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
func subtreeDefinition(c *cache.Reader, p position) (root, firstLeaf position, width uint64) {
	// maxRootHeight represents the max height of the tree, based on the width of base layer. This is used to prevent an
	// infinite loop.
	maxRootHeight := cache.RootHeightFromWidth(c.GetLayerReader(p.height).Width())
	for root = p.parent(); root.height < maxRootHeight; root = root.parent() {
		if layer := c.GetLayerReader(root.height); layer != nil {
			break
		}
	}
	subtreeHeight := root.height - p.height
	firstLeaf = position{
		index:  root.index << subtreeHeight,
		height: p.height,
	}
	return root, firstLeaf, 1 << subtreeHeight
}
