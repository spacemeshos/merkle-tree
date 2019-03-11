package merkle

import "errors"

type TreeCache struct {
	readers map[uint]NodeReader
	hash    HashFunc
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

// subtreeDefinition returns the definition (firstLeaf and root positions, width) for the minimal subtree whose
// base layer includes p and where the root is on a cached layer. If no cached layer exists above the base layer, the
// subtree will reach the root of the original tree.
func (c *TreeCache) subtreeDefinition(p position) (root, firstLeaf position, width uint64) {
	// maxRootHeight represents the max height of the tree, based on the width of base layer. This is used to prevent an
	// infinite loop.
	maxRootHeight := rootHeightFromWidth(c.readers[p.height].Width())
	for root = p.parent(); root.height < maxRootHeight; root = root.parent() {
		if _, found := c.readers[root.height]; found {
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

func (c *TreeCache) LeafReader() NodeReader {
	return c.readers[0]
}
