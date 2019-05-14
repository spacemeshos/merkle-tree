package merkle_test

import (
	"encoding/hex"
	"errors"
	"github.com/spacemeshos/merkle-tree/cache"
	"github.com/stretchr/testify/require"
	"io"
	"testing"
	"time"
)

/*

	8-leaf tree (1st 2 bytes of each node):

	+--------------------------------------------------+
	|                       89a0                       |
	|           ba94                    633b           |
	|     cb59        0094        bd50        fa67     |
	|  0000  0100  0200  0300  0400  0500  0600  0700  |
	+--------------------------------------------------+

*/

func TestGenerateProof(t *testing.T) {
	r := require.New(t)

	leavesToProve := setOf(0, 4, 7)

	cacheWriter := cache.NewWriter(cache.MinHeightPolicy(0), cache.MakeSliceReadWriterFactory())

	tree, _ := NewTreeBuilder().
		WithCacheWriter(cacheWriter).
		WithLeavesToProve(leavesToProve).
		Build()
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("89a0f1577268cc19b0a39c7a69f804fd140640c699585eb635ebb03c06154cce")
	root := tree.Root()
	r.Equal(expectedRoot, root)

	cacheReader, err := cacheWriter.GetReader()
	r.NoError(err)

	assertWidth(r, 8, cacheReader.GetLayerReader(0))
	assertWidth(r, 4, cacheReader.GetLayerReader(1))
	assertWidth(r, 2, cacheReader.GetLayerReader(2))

	var leaves, proof, expectedProof nodes
	sortedIndices, leaves, proof, err := GenerateProof(leavesToProve, cacheReader)
	r.NoError(err)

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof)

	var expectedLeaves nodes
	for _, i := range leavesToProve.AsSortedSlice() {
		expectedLeaves = append(expectedLeaves, NewNodeFromUint64(i))
	}
	r.EqualValues(expectedLeaves, leaves)
	r.EqualValues([]uint64{0, 4, 7}, sortedIndices)
}

func BenchmarkGenerateProof(b *testing.B) {
	const treeHeight = 23
	r := require.New(b)

	leavesToProve := make(set)

	cacheWriter := cache.NewWriter(
		cache.Combine(
			cache.MinHeightPolicy(7),
			cache.SpecificLayersPolicy(map[uint]bool{0: true})),
		cache.MakeSliceReadWriterFactory())

	for i := 0; i < 20; i++ {
		leavesToProve[uint64(i)*400000] = true
	}

	tree, _ := NewTreeBuilder().
		WithCacheWriter(cacheWriter).
		WithLeavesToProve(leavesToProve).
		Build()
	for i := uint64(0); i < 1<<treeHeight; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}

	cacheReader, err := cacheWriter.GetReader()
	r.NoError(err)

	assertWidth(r, 1<<treeHeight, cacheReader.GetLayerReader(0))

	var leaves, proof, expectedProof nodes

	start := time.Now()
	_, leaves, proof, err = GenerateProof(leavesToProve, cacheReader)
	r.NoError(err)
	b.Log(time.Since(start))

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof)

	var expectedLeaves nodes
	for _, i := range leavesToProve.AsSortedSlice() {
		expectedLeaves = append(expectedLeaves, NewNodeFromUint64(i))
	}
	r.EqualValues(expectedLeaves, leaves)

	/*
	   proving_test.go:88: 1.213317ms
	*/
}

func TestGenerateProofWithRoot(t *testing.T) {
	r := require.New(t)

	leavesToProve := setOf(0, 4, 7)

	cacheWriter := cache.NewWriter(cache.MinHeightPolicy(0), cache.MakeSliceReadWriterFactory())

	tree, _ := NewTreeBuilder().
		WithCacheWriter(cacheWriter).
		WithLeavesToProve(leavesToProve).
		Build()
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("89a0f1577268cc19b0a39c7a69f804fd140640c699585eb635ebb03c06154cce")
	root := tree.Root()
	r.Equal(expectedRoot, root)

	cacheReader, err := cacheWriter.GetReader()
	r.NoError(err)

	assertWidth(r, 8, cacheReader.GetLayerReader(0))
	assertWidth(r, 4, cacheReader.GetLayerReader(1))
	assertWidth(r, 2, cacheReader.GetLayerReader(2))
	assertWidth(r, 1, cacheReader.GetLayerReader(3))
	cacheRoot, err := cacheReader.GetLayerReader(3).ReadNext()
	r.NoError(err)
	r.Equal(cacheRoot, expectedRoot)

	var leaves, proof, expectedProof nodes
	sortedIndices, leaves, proof, err := GenerateProof(leavesToProve, cacheReader)
	r.NoError(err)

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof)

	var expectedLeaves nodes
	for _, i := range leavesToProve.AsSortedSlice() {
		expectedLeaves = append(expectedLeaves, NewNodeFromUint64(i))
	}
	r.EqualValues(expectedLeaves, leaves)
	r.EqualValues([]uint64{0, 4, 7}, sortedIndices)
}

func TestGenerateProofWithoutCache(t *testing.T) {
	r := require.New(t)
	leavesToProve := setOf(0, 4, 7)
	cacheWriter := cache.NewWriter(cache.SpecificLayersPolicy(map[uint]bool{0: true}), cache.MakeSliceReadWriterFactory())
	tree, _ := NewTreeBuilder().
		WithCacheWriter(cacheWriter).
		WithLeavesToProve(leavesToProve).
		Build()
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("89a0f1577268cc19b0a39c7a69f804fd140640c699585eb635ebb03c06154cce")
	root := tree.Root()
	r.Equal(expectedRoot, root)

	cacheReader, err := cacheWriter.GetReader()
	r.NoError(err)

	assertWidth(r, 8, cacheReader.GetLayerReader(0))

	var leaves, proof, expectedProof nodes
	sortedIndices, leaves, proof, err := GenerateProof(leavesToProve, cacheReader)
	r.NoError(err)

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof)

	var expectedLeaves nodes
	for _, i := range leavesToProve.AsSortedSlice() {
		expectedLeaves = append(expectedLeaves, NewNodeFromUint64(i))
	}
	r.EqualValues(expectedLeaves, leaves)
	r.EqualValues([]uint64{0, 4, 7}, sortedIndices)
}

func TestGenerateProofWithSingleLayerCache(t *testing.T) {
	r := require.New(t)
	leavesToProve := setOf(0, 4, 7)
	cacheWriter := cache.NewWriter(
		cache.SpecificLayersPolicy(map[uint]bool{0: true, 2: true}),
		cache.MakeSliceReadWriterFactory())
	tree, _ := NewTreeBuilder().
		WithCacheWriter(cacheWriter).
		WithLeavesToProve(leavesToProve).
		Build()
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("89a0f1577268cc19b0a39c7a69f804fd140640c699585eb635ebb03c06154cce")
	root := tree.Root()
	r.Equal(expectedRoot, root)

	cacheReader, err := cacheWriter.GetReader()
	r.NoError(err)

	assertWidth(r, 8, cacheReader.GetLayerReader(0))
	assertWidth(r, 2, cacheReader.GetLayerReader(2))

	var leaves, proof, expectedProof nodes
	sortedIndices, leaves, proof, err := GenerateProof(leavesToProve, cacheReader)
	r.NoError(err)

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof)

	var expectedLeaves nodes
	for _, i := range leavesToProve.AsSortedSlice() {
		expectedLeaves = append(expectedLeaves, NewNodeFromUint64(i))
	}
	r.EqualValues(expectedLeaves, leaves)
	r.EqualValues([]uint64{0, 4, 7}, sortedIndices)
}

func TestGenerateProofWithSingleLayerCache2(t *testing.T) {
	r := require.New(t)
	leavesToProve := setOf(0, 4, 7)
	cacheWriter := cache.NewWriter(
		cache.SpecificLayersPolicy(map[uint]bool{0: true, 1: true}),
		cache.MakeSliceReadWriterFactory())
	tree, _ := NewTreeBuilder().
		WithCacheWriter(cacheWriter).
		WithLeavesToProve(leavesToProve).
		Build()
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("89a0f1577268cc19b0a39c7a69f804fd140640c699585eb635ebb03c06154cce")
	root := tree.Root()
	r.Equal(expectedRoot, root)

	cacheReader, err := cacheWriter.GetReader()
	r.NoError(err)

	assertWidth(r, 8, cacheReader.GetLayerReader(0))
	assertWidth(r, 4, cacheReader.GetLayerReader(1))

	var leaves, proof, expectedProof nodes
	sortedIndices, leaves, proof, err := GenerateProof(leavesToProve, cacheReader)
	r.NoError(err)

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof)

	var expectedLeaves nodes
	for _, i := range leavesToProve.AsSortedSlice() {
		expectedLeaves = append(expectedLeaves, NewNodeFromUint64(i))
	}
	r.EqualValues(expectedLeaves, leaves)
	r.EqualValues([]uint64{0, 4, 7}, sortedIndices)
}

func TestGenerateProofWithSingleLayerCache3(t *testing.T) {
	r := require.New(t)
	leavesToProve := setOf(0)
	cacheWriter := cache.NewWriter(
		cache.SpecificLayersPolicy(map[uint]bool{0: true, 1: true}),
		cache.MakeSliceReadWriterFactory())
	tree, _ := NewTreeBuilder().
		WithCacheWriter(cacheWriter).
		WithLeavesToProve(leavesToProve).
		Build()
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("89a0f1577268cc19b0a39c7a69f804fd140640c699585eb635ebb03c06154cce")
	root := tree.Root()
	r.Equal(expectedRoot, root)

	cacheReader, err := cacheWriter.GetReader()
	r.NoError(err)

	assertWidth(r, 8, cacheReader.GetLayerReader(0))
	assertWidth(r, 4, cacheReader.GetLayerReader(1))

	var proof, expectedProof nodes
	_, _, proof, err = GenerateProof(leavesToProve, cacheReader)
	r.NoError(err)

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof)
}

func TestGenerateProofUnbalanced(t *testing.T) {
	r := require.New(t)
	leavesToProve := setOf(0, 4, 6)
	cacheWriter := cache.NewWriter(
		cache.SpecificLayersPolicy(map[uint]bool{0: true, 1: true, 2: true}),
		cache.MakeSliceReadWriterFactory())

	tree, _ := NewTreeBuilder().
		WithCacheWriter(cacheWriter).
		WithLeavesToProve(leavesToProve).
		Build()
	for i := uint64(0); i < 7; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}

	cacheReader, err := cacheWriter.GetReader()
	r.NoError(err)

	assertWidth(r, 7, cacheReader.GetLayerReader(0))
	assertWidth(r, 3, cacheReader.GetLayerReader(1))
	assertWidth(r, 1, cacheReader.GetLayerReader(2))

	var leaves, proof, expectedProof nodes
	sortedIndices, leaves, proof, err := GenerateProof(leavesToProve, cacheReader)
	r.NoError(err)

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof)

	var expectedLeaves nodes
	for _, i := range leavesToProve.AsSortedSlice() {
		expectedLeaves = append(expectedLeaves, NewNodeFromUint64(i))
	}
	r.EqualValues(expectedLeaves, leaves)
	r.EqualValues([]uint64{0, 4, 6}, sortedIndices)
}

func TestGenerateProofUnbalanced2(t *testing.T) {
	r := require.New(t)
	leavesToProve := setOf(0, 4)
	cacheWriter := cache.NewWriter(
		cache.SpecificLayersPolicy(map[uint]bool{0: true, 1: true, 2: true}),
		cache.MakeSliceReadWriterFactory())

	tree, _ := NewTreeBuilder().
		WithCacheWriter(cacheWriter).
		WithLeavesToProve(leavesToProve).
		Build()
	for i := uint64(0); i < 6; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}

	cacheReader, err := cacheWriter.GetReader()
	r.NoError(err)

	assertWidth(r, 6, cacheReader.GetLayerReader(0))
	assertWidth(r, 3, cacheReader.GetLayerReader(1))
	assertWidth(r, 1, cacheReader.GetLayerReader(2))

	var leaves, proof, expectedProof nodes
	sortedIndices, leaves, proof, err := GenerateProof(leavesToProve, cacheReader)
	r.NoError(err)

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof)

	var expectedLeaves nodes
	for _, i := range leavesToProve.AsSortedSlice() {
		expectedLeaves = append(expectedLeaves, NewNodeFromUint64(i))
	}
	r.EqualValues(expectedLeaves, leaves)
	r.EqualValues([]uint64{0, 4}, sortedIndices)
}

func TestGenerateProofUnbalanced3(t *testing.T) {
	r := require.New(t)
	leavesToProve := setOf(0)
	cacheWriter := cache.NewWriter(
		cache.SpecificLayersPolicy(map[uint]bool{0: true, 1: true, 2: true}),
		cache.MakeSliceReadWriterFactory())

	tree, _ := NewTreeBuilder().
		WithCacheWriter(cacheWriter).
		WithLeavesToProve(leavesToProve).
		Build()
	for i := uint64(0); i < 7; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}

	cacheReader, err := cacheWriter.GetReader()
	r.NoError(err)

	assertWidth(r, 7, cacheReader.GetLayerReader(0))
	assertWidth(r, 3, cacheReader.GetLayerReader(1))
	assertWidth(r, 1, cacheReader.GetLayerReader(2))

	var leaves, proof, expectedProof nodes
	sortedIndices, leaves, proof, err := GenerateProof(leavesToProve, cacheReader)
	r.NoError(err)

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof)

	var expectedLeaves nodes
	for _, i := range leavesToProve.AsSortedSlice() {
		expectedLeaves = append(expectedLeaves, NewNodeFromUint64(i))
	}
	r.EqualValues(expectedLeaves, leaves)
	r.EqualValues([]uint64{0}, sortedIndices)
}

type nodes [][]byte

func (n nodes) String() string {
	s := ""
	for _, v := range n {
		s += hex.EncodeToString(v[:2]) + " "
	}
	return s
}

var someError = errors.New("some error")

type seekErrorReader struct{}

// A compile time check to ensure that seekErrorReader fully implements LayerReadWriter.
var _ cache.LayerReadWriter = (*seekErrorReader)(nil)

func (seekErrorReader) Seek(index uint64) error            { return someError }
func (seekErrorReader) ReadNext() ([]byte, error)          { panic("implement me") }
func (seekErrorReader) Width() (uint64, error)             { return 3, nil }
func (seekErrorReader) Append(p []byte) (n int, err error) { panic("implement me") }
func (seekErrorReader) Flush() error                       { return nil }

type readErrorReader struct{}

// A compile time check to ensure that readErrorReader fully implements LayerReadWriter.
var _ cache.LayerReadWriter = (*readErrorReader)(nil)

func (readErrorReader) Seek(index uint64) error            { return nil }
func (readErrorReader) ReadNext() ([]byte, error)          { return nil, someError }
func (readErrorReader) Width() (uint64, error)             { return 8, nil }
func (readErrorReader) Append(p []byte) (n int, err error) { panic("implement me") }
func (readErrorReader) Flush() error                       { return nil }

type seekEOFReader struct{}

// A compile time check to ensure that seekEOFReader fully implements LayerReadWriter.
var _ cache.LayerReadWriter = (*seekEOFReader)(nil)

func (seekEOFReader) Seek(index uint64) error            { return io.EOF }
func (seekEOFReader) ReadNext() ([]byte, error)          { panic("implement me") }
func (seekEOFReader) Width() (uint64, error)             { return 1, nil }
func (seekEOFReader) Append(p []byte) (n int, err error) { panic("implement me") }
func (seekEOFReader) Flush() error                       { return nil }

type widthReader struct{ width uint64 }

// A compile time check to ensure that widthReader fully implements LayerReadWriter.
var _ cache.LayerReadWriter = (*widthReader)(nil)

func (r widthReader) Seek(index uint64) error            { return nil }
func (r widthReader) ReadNext() ([]byte, error)          { return nil, someError }
func (r widthReader) Width() (uint64, error)             { return r.width, nil }
func (r widthReader) Append(p []byte) (n int, err error) { panic("implement me") }
func (r widthReader) Flush() error                       { return nil }

func TestGetNode(t *testing.T) {
	r := require.New(t)

	cacheWriter := cache.NewWriter(cache.SpecificLayersPolicy(map[uint]bool{}), nil)
	cacheWriter.SetLayer(0, seekErrorReader{})

	cacheReader, err := cacheWriter.GetReader()
	r.NoError(err)

	nodePos := position{}
	node, err := GetNode(cacheReader, nodePos)

	r.Error(err)
	r.Equal("while seeking to Position <h: 0 i: 0> in cache: some error", err.Error())
	r.Nil(node)

}

func TestGetNode2(t *testing.T) {
	r := require.New(t)
	cacheWriter := cache.NewWriter(nil, nil)
	cacheWriter.SetLayer(0, readErrorReader{})

	cacheReader, err := cacheWriter.GetReader()
	r.NoError(err)
	nodePos := position{}
	node, err := GetNode(cacheReader, nodePos)

	r.Error(err)
	r.Equal("while reading from cache: some error", err.Error())
	r.Nil(node)
}

func TestGetNode3(t *testing.T) {
	r := require.New(t)
	cacheWriter := cache.NewWriter(nil, nil)
	cacheWriter.SetLayer(0, seekErrorReader{})
	cacheWriter.SetLayer(1, seekEOFReader{})

	cacheReader, err := cacheWriter.GetReader()
	r.NoError(err)
	nodePos := position{Height: 1}
	node, err := GetNode(cacheReader, nodePos)

	r.Error(err)
	r.Equal("while seeking to Position <h: 0 i: 0> in cache: some error", err.Error())
	r.Nil(node)
}

func TestGetNode4(t *testing.T) {
	r := require.New(t)
	cacheWriter := cache.NewWriter(nil, nil)
	cacheWriter.SetLayer(0, seekErrorReader{})
	cacheWriter.SetLayer(1, widthReader{width: 1})
	cacheWriter.SetLayer(2, seekEOFReader{})

	cacheReader, err := cacheWriter.GetReader()
	r.NoError(err)
	nodePos := position{Height: 2}
	node, err := GetNode(cacheReader, nodePos)

	r.Error(err)
	r.Equal("while calculating ephemeral node at Position <h: 1 i: 1>: while seeking to Position <h: 0 i: 10> in cache: some error", err.Error())
	r.Nil(node)
}

func TestGetNode5(t *testing.T) {
	r := require.New(t)
	cacheWriter := cache.NewWriter(nil, nil)
	cacheWriter.SetLayer(0, widthReader{width: 2})
	cacheWriter.SetLayer(1, seekEOFReader{})

	cacheReader, err := cacheWriter.GetReader()
	r.NoError(err)
	nodePos := position{Height: 1}
	node, err := GetNode(cacheReader, nodePos)

	r.Error(err)
	r.Equal("while traversing subtree for root: while reading a leaf: some error", err.Error())
	r.Nil(node)
}

func TestCache_ValidateStructure(t *testing.T) {
	r := require.New(t)
	cacheWriter := cache.NewWriter(nil, nil)
	cacheReader, err := cacheWriter.GetReader()

	r.Error(err)
	r.Equal("reader for base layer must be included", err.Error())
	r.Nil(cacheReader)
}
