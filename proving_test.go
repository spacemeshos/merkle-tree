package merkle

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

	tree := NewTreeBuilder().
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

	r.Equal(uint64(8), cacheReader.GetLayerReader(0).Width())
	r.Equal(uint64(4), cacheReader.GetLayerReader(1).Width())
	r.Equal(uint64(2), cacheReader.GetLayerReader(2).Width())

	var leaves, proof, expectedProof nodes
	leaves, proof, err = GenerateProof(leavesToProve, cacheReader)
	r.NoError(err)

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof)

	var expectedLeaves nodes
	for _, i := range leavesToProve.asSortedSlice() {
		expectedLeaves = append(expectedLeaves, NewNodeFromUint64(i))
	}
	r.EqualValues(expectedLeaves, leaves)
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

	tree := NewTreeBuilder().
		WithCacheWriter(cacheWriter).
		WithLeavesToProve(leavesToProve).
		Build()
	for i := uint64(0); i < 1<<treeHeight; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}

	cacheReader, err := cacheWriter.GetReader()
	r.NoError(err)

	r.Equal(uint64(1)<<treeHeight, cacheReader.GetLayerReader(0).Width())

	var leaves, proof, expectedProof nodes

	start := time.Now()
	leaves, proof, err = GenerateProof(leavesToProve, cacheReader)
	r.NoError(err)
	b.Log(time.Since(start))

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof)

	var expectedLeaves nodes
	for _, i := range leavesToProve.asSortedSlice() {
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

	tree := NewTreeBuilder().
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

	r.Equal(uint64(8), cacheReader.GetLayerReader(0).Width())
	r.Equal(uint64(4), cacheReader.GetLayerReader(1).Width())
	r.Equal(uint64(2), cacheReader.GetLayerReader(2).Width())
	r.Equal(uint64(1), cacheReader.GetLayerReader(3).Width())
	cacheRoot, err := cacheReader.GetLayerReader(3).ReadNext()
	r.NoError(err)
	r.Equal(cacheRoot, expectedRoot)

	var leaves, proof, expectedProof nodes
	leaves, proof, err = GenerateProof(leavesToProve, cacheReader)
	r.NoError(err)

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof)

	var expectedLeaves nodes
	for _, i := range leavesToProve.asSortedSlice() {
		expectedLeaves = append(expectedLeaves, NewNodeFromUint64(i))
	}
	r.EqualValues(expectedLeaves, leaves)
}

func TestGenerateProofWithoutCache(t *testing.T) {
	r := require.New(t)
	leavesToProve := setOf(0, 4, 7)
	cacheWriter := cache.NewWriter(cache.SpecificLayersPolicy(map[uint]bool{0: true}), cache.MakeSliceReadWriterFactory())
	tree := NewTreeBuilder().
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

	r.Equal(uint64(8), cacheReader.GetLayerReader(0).Width())

	var leaves, proof, expectedProof nodes
	leaves, proof, err = GenerateProof(leavesToProve, cacheReader)
	r.NoError(err)

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof)

	var expectedLeaves nodes
	for _, i := range leavesToProve.asSortedSlice() {
		expectedLeaves = append(expectedLeaves, NewNodeFromUint64(i))
	}
	r.EqualValues(expectedLeaves, leaves)
}

func TestGenerateProofWithSingleLayerCache(t *testing.T) {
	r := require.New(t)
	leavesToProve := setOf(0, 4, 7)
	cacheWriter := cache.NewWriter(
		cache.SpecificLayersPolicy(map[uint]bool{0: true, 2: true}),
		cache.MakeSliceReadWriterFactory())
	tree := NewTreeBuilder().
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

	r.Equal(uint64(8), cacheReader.GetLayerReader(0).Width())
	r.Equal(uint64(2), cacheReader.GetLayerReader(2).Width())

	var leaves, proof, expectedProof nodes
	leaves, proof, err = GenerateProof(leavesToProve, cacheReader)
	r.NoError(err)

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof)

	var expectedLeaves nodes
	for _, i := range leavesToProve.asSortedSlice() {
		expectedLeaves = append(expectedLeaves, NewNodeFromUint64(i))
	}
	r.EqualValues(expectedLeaves, leaves)
}

func TestGenerateProofWithSingleLayerCache2(t *testing.T) {
	r := require.New(t)
	leavesToProve := setOf(0, 4, 7)
	cacheWriter := cache.NewWriter(
		cache.SpecificLayersPolicy(map[uint]bool{0: true, 1: true}),
		cache.MakeSliceReadWriterFactory())
	tree := NewTreeBuilder().
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

	r.Equal(uint64(8), cacheReader.GetLayerReader(0).Width())
	r.Equal(uint64(4), cacheReader.GetLayerReader(1).Width())

	var leaves, proof, expectedProof nodes
	leaves, proof, err = GenerateProof(leavesToProve, cacheReader)
	r.NoError(err)

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof)

	var expectedLeaves nodes
	for _, i := range leavesToProve.asSortedSlice() {
		expectedLeaves = append(expectedLeaves, NewNodeFromUint64(i))
	}
	r.EqualValues(expectedLeaves, leaves)
}

func TestGenerateProofWithSingleLayerCache3(t *testing.T) {
	r := require.New(t)
	leavesToProve := setOf(0)
	cacheWriter := cache.NewWriter(
		cache.SpecificLayersPolicy(map[uint]bool{0: true, 1: true}),
		cache.MakeSliceReadWriterFactory())
	tree := NewTreeBuilder().
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

	r.Equal(uint64(8), cacheReader.GetLayerReader(0).Width())
	r.Equal(uint64(4), cacheReader.GetLayerReader(1).Width())

	var proof, expectedProof nodes
	_, proof, err = GenerateProof(leavesToProve, cacheReader)
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

	tree := NewTreeBuilder().
		WithCacheWriter(cacheWriter).
		WithLeavesToProve(leavesToProve).
		Build()
	for i := uint64(0); i < 7; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}

	cacheReader, err := cacheWriter.GetReader()
	r.NoError(err)

	r.Equal(uint64(7), cacheReader.GetLayerReader(0).Width())
	r.Equal(uint64(3), cacheReader.GetLayerReader(1).Width())
	r.Equal(uint64(1), cacheReader.GetLayerReader(2).Width())

	var leaves, proof, expectedProof nodes
	leaves, proof, err = GenerateProof(leavesToProve, cacheReader)
	r.NoError(err)

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof)

	var expectedLeaves nodes
	for _, i := range leavesToProve.asSortedSlice() {
		expectedLeaves = append(expectedLeaves, NewNodeFromUint64(i))
	}
	r.EqualValues(expectedLeaves, leaves)
}

func TestGenerateProofUnbalanced2(t *testing.T) {
	r := require.New(t)
	leavesToProve := setOf(0, 4)
	cacheWriter := cache.NewWriter(
		cache.SpecificLayersPolicy(map[uint]bool{0: true, 1: true, 2: true}),
		cache.MakeSliceReadWriterFactory())

	tree := NewTreeBuilder().
		WithCacheWriter(cacheWriter).
		WithLeavesToProve(leavesToProve).
		Build()
	for i := uint64(0); i < 6; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}

	cacheReader, err := cacheWriter.GetReader()
	r.NoError(err)

	r.Equal(uint64(6), cacheReader.GetLayerReader(0).Width())
	r.Equal(uint64(3), cacheReader.GetLayerReader(1).Width())
	r.Equal(uint64(1), cacheReader.GetLayerReader(2).Width())

	var leaves, proof, expectedProof nodes
	leaves, proof, err = GenerateProof(leavesToProve, cacheReader)
	r.NoError(err)

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof)

	var expectedLeaves nodes
	for _, i := range leavesToProve.asSortedSlice() {
		expectedLeaves = append(expectedLeaves, NewNodeFromUint64(i))
	}
	r.EqualValues(expectedLeaves, leaves)
}

func TestGenerateProofUnbalanced3(t *testing.T) {
	r := require.New(t)
	leavesToProve := setOf(0)
	cacheWriter := cache.NewWriter(
		cache.SpecificLayersPolicy(map[uint]bool{0: true, 1: true, 2: true}),
		cache.MakeSliceReadWriterFactory())

	tree := NewTreeBuilder().
		WithCacheWriter(cacheWriter).
		WithLeavesToProve(leavesToProve).
		Build()
	for i := uint64(0); i < 7; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}

	cacheReader, err := cacheWriter.GetReader()
	r.NoError(err)

	r.Equal(uint64(7), cacheReader.GetLayerReader(0).Width())
	r.Equal(uint64(3), cacheReader.GetLayerReader(1).Width())
	r.Equal(uint64(1), cacheReader.GetLayerReader(2).Width())

	var leaves, proof, expectedProof nodes
	leaves, proof, err = GenerateProof(leavesToProve, cacheReader)
	r.NoError(err)

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof)

	var expectedLeaves nodes
	for _, i := range leavesToProve.asSortedSlice() {
		expectedLeaves = append(expectedLeaves, NewNodeFromUint64(i))
	}
	r.EqualValues(expectedLeaves, leaves)
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

func (seekErrorReader) Seek(index uint64) error           { return someError }
func (seekErrorReader) ReadNext() ([]byte, error)         { panic("implement me") }
func (seekErrorReader) Width() uint64                     { return 3 }
func (seekErrorReader) Append(p []byte) (n int, err error) { panic("implement me") }

type readErrorReader struct{}

func (readErrorReader) Seek(index uint64) error           { return nil }
func (readErrorReader) ReadNext() ([]byte, error)         { return nil, someError }
func (readErrorReader) Width() uint64                     { return 8 }
func (readErrorReader) Append(p []byte) (n int, err error) { panic("implement me") }

type seekEOFReader struct{}

func (seekEOFReader) Seek(index uint64) error           { return io.EOF }
func (seekEOFReader) ReadNext() ([]byte, error)         { panic("implement me") }
func (seekEOFReader) Width() uint64                     { return 1 }
func (seekEOFReader) Append(p []byte) (n int, err error) { panic("implement me") }

type widthReader struct{ width uint64 }

func (r widthReader) Seek(index uint64) error           { return nil }
func (r widthReader) ReadNext() ([]byte, error)         { return nil, someError }
func (r widthReader) Width() uint64                     { return r.width }
func (r widthReader) Append(p []byte) (n int, err error) { panic("implement me") }

func TestGetNode(t *testing.T) {
	r := require.New(t)

	cacheWriter := cache.NewWriter(cache.SpecificLayersPolicy(map[uint]bool{}), nil)
	cacheWriter.SetLayer(0, seekErrorReader{})

	cacheReader, err := cacheWriter.GetReader()
	r.NoError(err)

	nodePos := position{}
	node, err := GetNode(cacheReader, nodePos)

	r.Error(err)
	r.Equal("while seeking to position <h: 0 i: 0> in cache: some error", err.Error())
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
	nodePos := position{height: 1}
	node, err := GetNode(cacheReader, nodePos)

	r.Error(err)
	r.Equal("while seeking to position <h: 0 i: 0> in cache: some error", err.Error())
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
	nodePos := position{height: 2}
	node, err := GetNode(cacheReader, nodePos)

	r.Error(err)
	r.Equal("while calculating ephemeral node at position <h: 1 i: 1>: while seeking to position <h: 0 i: 10> in cache: some error", err.Error())
	r.Nil(node)
}

func TestGetNode5(t *testing.T) {
	r := require.New(t)
	cacheWriter := cache.NewWriter(nil, nil)
	cacheWriter.SetLayer(0, widthReader{width: 2})
	cacheWriter.SetLayer(1, seekEOFReader{})

	cacheReader, err := cacheWriter.GetReader()
	r.NoError(err)
	nodePos := position{height: 1}
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
