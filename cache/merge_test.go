package cache

import (
	"encoding/binary"
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/merkle-tree/cache/readwriters"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMerge(t *testing.T) {
	r := require.New(t)

	readers := make([]*Reader, 3)
	readers[0] = &Reader{&cache{layers: make(map[uint]LayerReadWriter)}}
	readers[1] = &Reader{&cache{layers: make(map[uint]LayerReadWriter)}}
	readers[2] = &Reader{&cache{layers: make(map[uint]LayerReadWriter)}}

	// Create 9 nodes.
	nodes := genNodes(9)

	// Split the nodes into 3 layers.
	splitLayer := make([]LayerReadWriter, 3)
	splitLayer[0] = &readwriters.SliceReadWriter{}
	splitLayer[1] = &readwriters.SliceReadWriter{}
	splitLayer[2] = &readwriters.SliceReadWriter{}
	_, _ = splitLayer[0].Append(nodes[0])
	_, _ = splitLayer[0].Append(nodes[1])
	_, _ = splitLayer[0].Append(nodes[2])
	_, _ = splitLayer[1].Append(nodes[3])
	_, _ = splitLayer[1].Append(nodes[4])
	_, _ = splitLayer[1].Append(nodes[5])
	_, _ = splitLayer[2].Append(nodes[6])
	_, _ = splitLayer[2].Append(nodes[7])
	_, _ = splitLayer[2].Append(nodes[8])

	// Assign the split layer into 3 different readers on height 0.
	readers[0].cache.layers[0] = splitLayer[0]
	readers[1].cache.layers[0] = splitLayer[1]
	readers[2].cache.layers[0] = splitLayer[2]

	var caches []CacheReader
	for _, reader := range readers {
		caches = append(caches, CacheReader(reader))
	}
	cache, err := Merge(caches)
	r.NoError(err)

	// Verify the split layers group.
	layer := cache.GetLayerReader(0)
	width, err := layer.Width()
	r.NoError(err)
	r.Equal(width, uint64(len(nodes)))

	// Iterate over the layer.
	for _, node := range nodes {
		val, err := layer.ReadNext()
		r.NoError(err)
		r.Equal(val, node)
	}
}

func TestMergeFailure1(t *testing.T) {
	r := require.New(t)

	readers := make([]*Reader, 1)
	readers[0] = &Reader{&cache{layers: make(map[uint]LayerReadWriter)}}

	var caches []CacheReader
	for _, reader := range readers {
		caches = append(caches, CacheReader(reader))
	}
	_, err := Merge(caches)
	r.Equal("number of caches must be at least 2", err.Error())
}

func TestMergeFailure2(t *testing.T) {
	r := require.New(t)

	readers := make([]*Reader, 2)
	readers[0] = &Reader{&cache{layers: make(map[uint]LayerReadWriter)}}
	readers[1] = &Reader{&cache{layers: make(map[uint]LayerReadWriter)}}

	readers[0].cache.layers[0] = &readwriters.SliceReadWriter{}

	var caches []CacheReader
	for _, reader := range readers {
		caches = append(caches, CacheReader(reader))
	}
	_, err := Merge(caches)
	r.Equal("number of layers per height mismatch", err.Error())
}

func TestMergeAndBuildTopCache(t *testing.T) {
	r := require.New(t)

	// Create 4 trees.
	cacheReaders := make([]CacheReader, 4)
	for i := 0; i < 4; i++ {
		cacheWriter := NewWriter(MinHeightPolicy(0), MakeSliceReadWriterFactory())
		tree, err := merkle.NewCachingTree(cacheWriter)
		r.NoError(err)
		for i := uint64(0); i < 8; i++ {
			err := tree.AddLeaf(NewNodeFromUint64(i))
			r.NoError(err)
		}

		cacheReader, err := cacheWriter.GetReader()
		r.NoError(err)

		assertWidth(r, 8, cacheReader.GetLayerReader(0))
		assertWidth(r, 4, cacheReader.GetLayerReader(1))
		assertWidth(r, 2, cacheReader.GetLayerReader(2))
		assertWidth(r, 1, cacheReader.GetLayerReader(3))
		cacheRoot, err := cacheReader.GetLayerReader(3).ReadNext()
		r.NoError(err)
		r.Equal(cacheRoot, tree.Root())
		err = cacheReader.GetLayerReader(3).Seek(0) // Reset position.
		r.NoError(err)

		cacheReaders[i] = cacheReader
	}

	// Merge caches, and verify that the upper subtree is missing.
	cacheReader, err := Merge(cacheReaders)
	r.NoError(err)
	assertWidth(r, 32, cacheReader.GetLayerReader(0))
	assertWidth(r, 16, cacheReader.GetLayerReader(1))
	assertWidth(r, 8, cacheReader.GetLayerReader(2))
	assertWidth(r, 4, cacheReader.GetLayerReader(3))
	r.Nil(cacheReader.GetLayerReader(4))
	r.Nil(cacheReader.GetLayerReader(5))

	// Create the upper subtree.
	cacheReader, root, err := BuildTop(cacheReader)
	r.NoError(err)
	assertWidth(r, 32, cacheReader.GetLayerReader(0))
	assertWidth(r, 16, cacheReader.GetLayerReader(1))
	assertWidth(r, 8, cacheReader.GetLayerReader(2))
	assertWidth(r, 4, cacheReader.GetLayerReader(3))
	assertWidth(r, 2, cacheReader.GetLayerReader(4))
	assertWidth(r, 1, cacheReader.GetLayerReader(5))

	// Compare the cache root with the root received from BuildTop.
	cacheRoot, err := cacheReader.GetLayerReader(5).ReadNext()
	r.NoError(err)
	r.Equal(cacheRoot, root)
	err = cacheReader.GetLayerReader(5).Seek(0) // Reset position.
	r.NoError(err)
}

func TestMergeAndBuildTop(t *testing.T) {
	r := require.New(t)

	// Create 32 nodes.
	nodes := genNodes(32)

	// Add the nodes as leaves to one tree and save its root.
	cacheWriter := NewWriter(MinHeightPolicy(0), MakeSliceReadWriterFactory())
	tree, err := merkle.NewCachingTree(cacheWriter)
	r.NoError(err)
	for i := 0; i < len(nodes); i++ {
		err := tree.AddLeaf(NewNodeFromUint64(uint64(i)))
		r.NoError(err)
	}
	treeRoot := tree.Root()

	// Add the nodes as leaves to 4 separate trees.
	cacheWriters := make([]CacheWriter, 4)
	cacheReaders := make([]CacheReader, 4)
	trees := make([]*merkle.Tree, 4)
	for i := 0; i < 4; i++ {
		cacheWriter := NewWriter(MinHeightPolicy(0), MakeSliceReadWriterFactory())
		tree, err := merkle.NewCachingTree(cacheWriter)
		r.NoError(err)

		cacheWriters[i] = cacheWriter
		trees[i] = tree
	}
	for i := 0; i < len(nodes); i++ {
		err := trees[i/8].AddLeaf(NewNodeFromUint64(uint64(i)))
		r.NoError(err)
	}
	for i := 0; i < 4; i++ {
		reader, err := cacheWriters[i].GetReader()
		r.NoError(err)
		cacheReaders[i] = reader
	}

	// Merge caches.
	cacheReader, err := Merge(cacheReaders)
	r.NoError(err)
	r.NotNil(cacheReader)

	// Create the upper subtree.
	cacheReader, mergeRoot, err := BuildTop(cacheReader)
	r.NoError(err)
	r.NotNil(cacheReader)

	// Verify that the 4 trees merge root is the same as the main tree root.
	r.Equal(mergeRoot, treeRoot)
}

// -- FAILING --
func TestMergeAndBuildTopUnbalanced(t *testing.T) {
	r := require.New(t)

	// Create 29 nodes.
	nodes := genNodes(29)

	// Add the nodes as leaves to one tree and save its root.
	cacheWriter := NewWriter(MinHeightPolicy(0), MakeSliceReadWriterFactory())
	tree, err := merkle.NewCachingTree(cacheWriter)
	r.NoError(err)
	for i := 0; i < len(nodes); i++ {
		err := tree.AddLeaf(NewNodeFromUint64(uint64(i)))
		r.NoError(err)
	}
	treeRoot := tree.Root()

	// Add the nodes as leaves to 4 separate trees.
	cacheWriters := make([]CacheWriter, 4)
	cacheReaders := make([]CacheReader, 4)
	trees := make([]*merkle.Tree, 4)
	for i := 0; i < 4; i++ {
		cacheWriter := NewWriter(MinHeightPolicy(0), MakeSliceReadWriterFactory())
		tree, err := merkle.NewCachingTree(cacheWriter)
		r.NoError(err)

		cacheWriters[i] = cacheWriter
		trees[i] = tree
	}
	for i := 0; i < len(nodes); i++ {
		err := trees[i/8].AddLeaf(NewNodeFromUint64(uint64(i)))
		r.NoError(err)
	}
	for i := 0; i < 4; i++ {
		reader, err := cacheWriters[i].GetReader()
		r.NoError(err)
		cacheReaders[i] = reader
	}

	// Merge caches.
	cacheReader, err := Merge(cacheReaders)
	r.NoError(err)
	r.NotNil(cacheReader)

	// Create the upper subtree.
	cacheReader, mergeRoot, err := BuildTop(cacheReader)
	r.NoError(err)
	r.NotNil(cacheReader)

	// Verify that the 4 trees merge root is the same as the main tree root.
	r.Equal(mergeRoot, treeRoot)
}

func genNodes(num int) [][]byte {
	nodes := make([][]byte, num)
	for i := 0; i < num; i++ {
		nodes[i] = NewNodeFromUint64(uint64(i))
	}
	return nodes
}

func NewNodeFromUint64(i uint64) []byte {
	b := make([]byte, NodeSize)
	binary.LittleEndian.PutUint64(b, i)
	return b
}

func assertWidth(r *require.Assertions, expectedWidth int, layerReader LayerReader) {
	width, err := layerReader.Width()
	r.NoError(err)
	r.Equal(uint64(expectedWidth), width)
}
