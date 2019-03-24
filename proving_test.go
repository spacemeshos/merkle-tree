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

	leavesToProve := []uint64{0, 4, 7}

	treeCache := cache.NewWriterWithLayerFactories([]cache.LayerFactory{cache.MakeMemoryReadWriterFactory(0)})

	tree := NewTreeBuilder().
		WithCache(treeCache).
		WithLeavesToProve(leavesToProve).
		Build()
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("89a0f1577268cc19b0a39c7a69f804fd140640c699585eb635ebb03c06154cce")
	root := tree.Root()
	r.Equal(expectedRoot, root)

	reader, err := treeCache.GetReader()
	r.NoError(err)

	r.Equal(uint64(8), reader.GetLayerReader(0).Width())
	r.Equal(uint64(4), reader.GetLayerReader(1).Width())
	r.Equal(uint64(2), reader.GetLayerReader(2).Width())

	var proof, expectedProof nodes
	proof, err = GenerateProof(leavesToProve, reader)
	r.NoError(err)

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof, "actual")
}

func BenchmarkGenerateProof(b *testing.B) {
	const treeHeight = 23
	r := require.New(b)

	var leavesToProve []uint64

	treeCache := cache.NewWriterWithLayerFactories([]cache.LayerFactory{
		cache.MakeMemoryReadWriterFactoryForLayers([]uint{0}),
		cache.MakeMemoryReadWriterFactory(7),
	})

	for i := 0; i < 20; i++ {
		leavesToProve = append(leavesToProve, uint64(i)*400000)
	}

	tree := NewTreeBuilder().
		WithCache(treeCache).
		WithLeavesToProve(leavesToProve).
		Build()
	for i := uint64(0); i < 1<<treeHeight; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}

	reader, err := treeCache.GetReader()
	r.NoError(err)

	r.Equal(uint64(1)<<treeHeight, reader.GetLayerReader(0).Width())

	var proof, expectedProof nodes

	start := time.Now()
	proof, err = GenerateProof(leavesToProve, reader)
	r.NoError(err)
	b.Log(time.Since(start))

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof, "actual")

	/*
	   proving_test.go:88: 1.213317ms
	*/
}

func TestGenerateProofWithRoot(t *testing.T) {
	r := require.New(t)

	leavesToProve := []uint64{0, 4, 7}

	treeCache := cache.NewWriterWithLayerFactories([]cache.LayerFactory{cache.MakeMemoryReadWriterFactory(0)})

	tree := NewTreeBuilder().
		WithCache(treeCache).
		WithLeavesToProve(leavesToProve).
		Build()
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("89a0f1577268cc19b0a39c7a69f804fd140640c699585eb635ebb03c06154cce")
	root := tree.Root()
	r.Equal(expectedRoot, root)

	reader, err := treeCache.GetReader()
	r.NoError(err)

	r.Equal(uint64(8), reader.GetLayerReader(0).Width())
	r.Equal(uint64(4), reader.GetLayerReader(1).Width())
	r.Equal(uint64(2), reader.GetLayerReader(2).Width())
	r.Equal(uint64(1), reader.GetLayerReader(3).Width())
	cacheRoot, err := reader.GetLayerReader(3).ReadNext()
	r.NoError(err)
	r.Equal(cacheRoot, expectedRoot)

	var proof, expectedProof nodes
	proof, err = GenerateProof(leavesToProve, reader)
	r.NoError(err)

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof, "actual")
}

func TestGenerateProofWithoutCache(t *testing.T) {
	r := require.New(t)
	leavesToProve := []uint64{0, 4, 7}
	treeCache := cache.NewWriterWithLayerFactories(
		[]cache.LayerFactory{cache.MakeMemoryReadWriterFactoryForLayers([]uint{0})},
	)
	tree := NewTreeBuilder().
		WithCache(treeCache).
		WithLeavesToProve(leavesToProve).
		Build()
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("89a0f1577268cc19b0a39c7a69f804fd140640c699585eb635ebb03c06154cce")
	root := tree.Root()
	r.Equal(expectedRoot, root)

	reader, err := treeCache.GetReader()
	r.NoError(err)

	r.Equal(uint64(8), reader.GetLayerReader(0).Width())

	var proof, expectedProof nodes
	proof, err = GenerateProof(leavesToProve, reader)
	r.NoError(err)

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof, "actual")
}

func TestGenerateProofWithSingleLayerCache(t *testing.T) {
	r := require.New(t)
	leavesToProve := []uint64{0, 4, 7}
	treeCache := cache.NewWriterWithLayerFactories(
		[]cache.LayerFactory{cache.MakeMemoryReadWriterFactoryForLayers([]uint{0, 2})},
	)
	tree := NewTreeBuilder().
		WithCache(treeCache).
		WithLeavesToProve(leavesToProve).
		Build()
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("89a0f1577268cc19b0a39c7a69f804fd140640c699585eb635ebb03c06154cce")
	root := tree.Root()
	r.Equal(expectedRoot, root)

	reader, err := treeCache.GetReader()
	r.NoError(err)

	r.Equal(uint64(8), reader.GetLayerReader(0).Width())
	r.Equal(uint64(2), reader.GetLayerReader(2).Width())

	var proof, expectedProof nodes
	proof, err = GenerateProof(leavesToProve, reader)
	r.NoError(err)

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof)
}

func TestGenerateProofWithSingleLayerCache2(t *testing.T) {
	r := require.New(t)
	leavesToProve := []uint64{0, 4, 7}
	treeCache := cache.NewWriterWithLayerFactories(
		[]cache.LayerFactory{cache.MakeMemoryReadWriterFactoryForLayers([]uint{0, 1})},
	)
	tree := NewTreeBuilder().
		WithCache(treeCache).
		WithLeavesToProve(leavesToProve).
		Build()
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("89a0f1577268cc19b0a39c7a69f804fd140640c699585eb635ebb03c06154cce")
	root := tree.Root()
	r.Equal(expectedRoot, root)

	reader, err := treeCache.GetReader()
	r.NoError(err)

	r.Equal(uint64(8), reader.GetLayerReader(0).Width())
	r.Equal(uint64(4), reader.GetLayerReader(1).Width())

	var proof, expectedProof nodes
	proof, err = GenerateProof(leavesToProve, reader)
	r.NoError(err)

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof)
}

func TestGenerateProofWithSingleLayerCache3(t *testing.T) {
	r := require.New(t)
	leavesToProve := []uint64{0}
	treeCache := cache.NewWriterWithLayerFactories(
		[]cache.LayerFactory{cache.MakeMemoryReadWriterFactoryForLayers([]uint{0, 1})},
	)
	tree := NewTreeBuilder().
		WithCache(treeCache).
		WithLeavesToProve(leavesToProve).
		Build()
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("89a0f1577268cc19b0a39c7a69f804fd140640c699585eb635ebb03c06154cce")
	root := tree.Root()
	r.Equal(expectedRoot, root)

	reader, err := treeCache.GetReader()
	r.NoError(err)

	r.Equal(uint64(8), reader.GetLayerReader(0).Width())
	r.Equal(uint64(4), reader.GetLayerReader(1).Width())

	var proof, expectedProof nodes
	proof, err = GenerateProof(leavesToProve, reader)
	r.NoError(err)

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof)
}

func TestGenerateProofUnbalanced(t *testing.T) {
	r := require.New(t)
	leavesToProve := []uint64{0, 4, 6}
	treeCache := cache.NewWriterWithLayerFactories(
		[]cache.LayerFactory{cache.MakeMemoryReadWriterFactoryForLayers([]uint{0, 1, 2})},
	)

	tree := NewTreeBuilder().
		WithCache(treeCache).
		WithLeavesToProve(leavesToProve).
		Build()
	for i := uint64(0); i < 7; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}

	reader, err := treeCache.GetReader()
	r.NoError(err)

	r.Equal(uint64(7), reader.GetLayerReader(0).Width())
	r.Equal(uint64(3), reader.GetLayerReader(1).Width())
	r.Equal(uint64(1), reader.GetLayerReader(2).Width())

	var proof, expectedProof nodes
	proof, err = GenerateProof(leavesToProve, reader)
	r.NoError(err)

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof)
}

func TestGenerateProofUnbalanced2(t *testing.T) {
	r := require.New(t)
	leavesToProve := []uint64{0, 4}
	treeCache := cache.NewWriterWithLayerFactories(
		[]cache.LayerFactory{cache.MakeMemoryReadWriterFactoryForLayers([]uint{0, 1, 2})},
	)

	tree := NewTreeBuilder().
		WithCache(treeCache).
		WithLeavesToProve(leavesToProve).
		Build()
	for i := uint64(0); i < 6; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}

	reader, err := treeCache.GetReader()
	r.NoError(err)

	r.Equal(uint64(6), reader.GetLayerReader(0).Width())
	r.Equal(uint64(3), reader.GetLayerReader(1).Width())
	r.Equal(uint64(1), reader.GetLayerReader(2).Width())

	var proof, expectedProof nodes
	proof, err = GenerateProof(leavesToProve, reader)
	r.NoError(err)

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof)
}

func TestGenerateProofUnbalanced3(t *testing.T) {
	r := require.New(t)
	leavesToProve := []uint64{0}
	treeCache := cache.NewWriterWithLayerFactories(
		[]cache.LayerFactory{cache.MakeMemoryReadWriterFactoryForLayers([]uint{0, 1, 2})},
	)

	tree := NewTreeBuilder().
		WithCache(treeCache).
		WithLeavesToProve(leavesToProve).
		Build()
	for i := uint64(0); i < 7; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}

	reader, err := treeCache.GetReader()
	r.NoError(err)

	r.Equal(uint64(7), reader.GetLayerReader(0).Width())
	r.Equal(uint64(3), reader.GetLayerReader(1).Width())
	r.Equal(uint64(1), reader.GetLayerReader(2).Width())

	var proof, expectedProof nodes
	proof, err = GenerateProof(leavesToProve, reader)
	r.NoError(err)

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof)
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
func (seekErrorReader) Write(p []byte) (n int, err error) { panic("implement me") }

type readErrorReader struct{}

func (readErrorReader) Seek(index uint64) error           { return nil }
func (readErrorReader) ReadNext() ([]byte, error)         { return nil, someError }
func (readErrorReader) Width() uint64                     { return 8 }
func (readErrorReader) Write(p []byte) (n int, err error) { panic("implement me") }

type seekEOFReader struct{}

func (seekEOFReader) Seek(index uint64) error           { return io.EOF }
func (seekEOFReader) ReadNext() ([]byte, error)         { panic("implement me") }
func (seekEOFReader) Width() uint64                     { return 1 }
func (seekEOFReader) Write(p []byte) (n int, err error) { panic("implement me") }

type widthReader struct{ width uint64 }

func (r widthReader) Seek(index uint64) error           { return nil }
func (r widthReader) ReadNext() ([]byte, error)         { return nil, someError }
func (r widthReader) Width() uint64                     { return r.width }
func (r widthReader) Write(p []byte) (n int, err error) { panic("implement me") }

func TestGetNode(t *testing.T) {
	r := require.New(t)

	treeCache := cache.NewWriterWithLayerFactories(
		[]cache.LayerFactory{cache.MakeSpecificLayerFactory(0, seekErrorReader{})},
	)
	treeCache.GetLayerWriter(0) // this uses the factory to produce the layer cache

	reader, err := treeCache.GetReader()
	r.NoError(err)

	nodePos := position{}
	node, err := GetNode(reader, nodePos)

	r.Error(err)
	r.Equal("while seeking to position <h: 0 i: 0> in cache: some error", err.Error())
	r.Nil(node)

}

func TestGetNode2(t *testing.T) {
	r := require.New(t)
	treeCache := cache.NewWriterWithLayerFactories([]cache.LayerFactory{
		cache.MakeSpecificLayerFactory(0, readErrorReader{}),
	})
	treeCache.GetLayerWriter(0) // this uses the factory to produce the layer cache

	reader, err := treeCache.GetReader()
	r.NoError(err)
	nodePos := position{}
	node, err := GetNode(reader, nodePos)

	r.Error(err)
	r.Equal("while reading from cache: some error", err.Error())
	r.Nil(node)
}

func TestGetNode3(t *testing.T) {
	r := require.New(t)
	treeCache := cache.NewWriterWithLayerFactories([]cache.LayerFactory{
		cache.MakeSpecificLayerFactory(0, seekErrorReader{}),
		cache.MakeSpecificLayerFactory(1, seekEOFReader{}),
	})
	treeCache.GetLayerWriter(0) // this uses the factory to produce the layer cache
	treeCache.GetLayerWriter(1) // this uses the factory to produce the layer cache

	reader, err := treeCache.GetReader()
	r.NoError(err)
	nodePos := position{height: 1}
	node, err := GetNode(reader, nodePos)

	r.Error(err)
	r.Equal("while seeking to position <h: 0 i: 0> in cache: some error", err.Error())
	r.Nil(node)
}

func TestGetNode4(t *testing.T) {
	r := require.New(t)
	treeCache := cache.NewWriterWithLayerFactories([]cache.LayerFactory{
		cache.MakeSpecificLayerFactory(0, seekErrorReader{}),
		cache.MakeSpecificLayerFactory(1, widthReader{width: 1}),
		cache.MakeSpecificLayerFactory(2, seekEOFReader{}),
	})
	treeCache.GetLayerWriter(0) // this uses the factory to produce the layer cache
	treeCache.GetLayerWriter(1) // this uses the factory to produce the layer cache
	treeCache.GetLayerWriter(2) // this uses the factory to produce the layer cache

	reader, err := treeCache.GetReader()
	r.NoError(err)
	nodePos := position{height: 2}
	node, err := GetNode(reader, nodePos)

	r.Error(err)
	r.Equal("while calculating ephemeral node at position <h: 1 i: 1>: while seeking to position <h: 0 i: 10> in cache: some error", err.Error())
	r.Nil(node)
}

func TestGetNode5(t *testing.T) {
	r := require.New(t)
	treeCache := cache.NewWriterWithLayerFactories([]cache.LayerFactory{
		cache.MakeSpecificLayerFactory(0, widthReader{width: 2}),
		cache.MakeSpecificLayerFactory(1, seekEOFReader{}),
	})
	treeCache.GetLayerWriter(0) // this uses the factory to produce the layer cache
	treeCache.GetLayerWriter(1) // this uses the factory to produce the layer cache

	reader, err := treeCache.GetReader()
	r.NoError(err)
	nodePos := position{height: 1}
	node, err := GetNode(reader, nodePos)

	r.Error(err)
	r.Equal("while traversing subtree for root: while reading a leaf: some error", err.Error())
	r.Nil(node)
}

func TestCache_ValidateStructure(t *testing.T) {
	r := require.New(t)
	treeCache := cache.NewWriterWithLayerFactories([]cache.LayerFactory{})
	reader, err := treeCache.GetReader()

	r.Error(err)
	r.Equal("reader for base layer must be included", err.Error())
	r.Nil(reader)
}
