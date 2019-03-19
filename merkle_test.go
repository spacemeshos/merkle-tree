package merkle

import (
	"encoding/binary"
	"encoding/hex"
	"github.com/spacemeshos/merkle-tree/cache"
	"github.com/stretchr/testify/require"
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

func TestNewTree(t *testing.T) {
	r := require.New(t)
	tree := NewTree()
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("89a0f1577268cc19b0a39c7a69f804fd140640c699585eb635ebb03c06154cce")
	root := tree.Root()
	r.Equal(expectedRoot, root)
}

func concatLeaves(lChild, rChild []byte) []byte {
	if len(lChild) == NodeSize {
		lChild = lChild[:1]
	}
	if len(rChild) == NodeSize {
		rChild = rChild[:1]
	}
	return append(lChild, rChild...)
}

func TestNewTreeWithMinHeightEqual(t *testing.T) {
	r := require.New(t)
	tree := NewTreeBuilder().WithHashFunc(concatLeaves).WithMinHeight(3).Build()
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("0001020304050607")
	root := tree.Root()
	r.Equal(expectedRoot, root)
}

func TestNewTreeWithMinHeightGreater(t *testing.T) {
	r := require.New(t)
	tree := NewTreeBuilder().WithHashFunc(concatLeaves).WithMinHeight(4).Build()
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	// An 8-leaf tree is 3 layers high, so setting a minHeight of 4 means we need to add a "padding node" to the root.
	expectedRoot, _ := NewNodeFromHex("000102030405060700")
	root := tree.Root()
	r.Equal(expectedRoot, root)
}

func TestNewTreeWithMinHeightGreater2(t *testing.T) {
	r := require.New(t)
	tree := NewTreeBuilder().WithHashFunc(concatLeaves).WithMinHeight(5).Build()
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	// An 8-leaf tree is 3 layers high, so setting a minHeight of 5 means we need to add two "padding nodes" to the root.
	expectedRoot, _ := NewNodeFromHex("00010203040506070000")
	root := tree.Root()
	r.Equal(expectedRoot, root)
}

func TestNewTreeUnbalanced(t *testing.T) {
	r := require.New(t)
	tree := NewTree()
	for i := uint64(0); i < 9; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("cb71c80ee780788eedb819ec125a41e0cde57bd0955cdd3157ca363193ab5ff1")
	root := tree.Root()
	r.Equal(expectedRoot, root)
}

func TestNewTreeUnbalanced2(t *testing.T) {
	r := require.New(t)
	tree := NewTree()
	for i := uint64(0); i < 10; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("59f32a43534fe4c4c0966421aef624267cdf65bd11f74998c60f27c7caccb12d")
	root := tree.Root()
	r.Equal(expectedRoot, root)
}

func TestNewTreeUnbalanced3(t *testing.T) {
	r := require.New(t)
	tree := NewTree()
	for i := uint64(0); i < 15; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("b9746fb884ed07041c5cbb3bb5526e1383928e832a8385e08db995966889b5a8")
	root := tree.Root()
	r.Equal(expectedRoot, root)
}

func TestNewTreeUnbalancedProof(t *testing.T) {
	r := require.New(t)

	leavesToProve := []uint64{0, 4, 7}

	treeCache := cache.NewCacheWithLayerFactories([]cache.LayerFactory{cache.MakeMemoryReadWriterFactory(0)})

	tree := NewTreeBuilder().
		WithLeavesToProve(leavesToProve).
		WithCache(treeCache).
		Build()
	for i := uint64(0); i < 10; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("59f32a43534fe4c4c0966421aef624267cdf65bd11f74998c60f27c7caccb12d")
	root := tree.Root()
	r.Equal(expectedRoot, root)

	r.Equal(uint64(10), treeCache.GetLayerReader(0).Width())
	r.Equal(uint64(5), treeCache.GetLayerReader(1).Width())
	r.Equal(uint64(2), treeCache.GetLayerReader(2).Width())
	r.Equal(uint64(1), treeCache.GetLayerReader(3).Width())

	cacheRoot, err := treeCache.GetLayerReader(3).ReadNext()
	r.NoError(err)
	r.NotEqual(cacheRoot, expectedRoot)

	expectedProof := make([][]byte, 5)
	expectedProof[0], _ = NewNodeFromHex("0100000000000000000000000000000000000000000000000000000000000000")
	expectedProof[1], _ = NewNodeFromHex("0094579cfc7b716038d416a311465309bea202baa922b224a7b08f01599642fb")
	expectedProof[2], _ = NewNodeFromHex("0500000000000000000000000000000000000000000000000000000000000000")
	expectedProof[3], _ = NewNodeFromHex("0600000000000000000000000000000000000000000000000000000000000000")
	expectedProof[4], _ = NewNodeFromHex("bc68417a8495de6e22d95b980fca5a1183f29eff0e2a9b7ddde91ed5bcbea952")

	var proof nodes
	proof = tree.Proof()
	r.EqualValues(expectedProof, proof)
}

func BenchmarkNewTree(b *testing.B) {
	var size uint64 = 1 << 28
	tree := NewTree()
	for i := uint64(0); i < size; i++ {
		_ = tree.AddLeaf(NewNodeFromUint64(i))
	}
	/*
		goos: darwin
		goarch: amd64
		pkg: github.com/spacemeshos/merkle-tree
		BenchmarkNewTree-8   	       1	125055682774 ns/op
		PASS
	*/
}

func BenchmarkNewTreeSmall(b *testing.B) {
	var size uint64 = 1 << 23
	start := time.Now()
	tree := NewTree()
	for i := uint64(0); i < size; i++ {
		_ = tree.AddLeaf(NewNodeFromUint64(i))
	}
	b.Log(time.Since(start))
	/*
	   merkle_test.go:72: 3.700763631s
	*/
}

func BenchmarkNewTreeNoHashing(b *testing.B) {
	var size uint64 = 1 << 28
	tree := NewTree()
	for i := uint64(0); i < size; i++ {
		_ = tree.AddLeaf(NewNodeFromUint64(i))
	}
	/*
		goos: darwin
		goarch: amd64
		pkg: github.com/spacemeshos/merkle-tree
		BenchmarkNewTreeNoHashing-8   	       1	14668889972 ns/op
		BenchmarkNewTreeNoHashing-8   	       1	15791579912 ns/op
		PASS
	*/
}

/*
	28 layer tree takes 125 seconds to construct. Overhead (no hashing) is 15.5 seconds. Net: 109.5 seconds.
	(8.5GB @ 32b leaves) => x30 256GB => 55 minutes for hashing, 8 minutes overhead.

	Reading 256GB from a magnetic disk should take ~30 minutes.
*/

// Proving tree tests

func TestNewProvingTree(t *testing.T) {
	r := require.New(t)
	tree := NewProvingTree([]uint64{4})
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("89a0f1577268cc19b0a39c7a69f804fd140640c699585eb635ebb03c06154cce")
	root := tree.Root()
	r.Equal(expectedRoot, root)

	expectedProof := make([][]byte, 3)
	expectedProof[0], _ = NewNodeFromHex("0500000000000000000000000000000000000000000000000000000000000000")
	expectedProof[1], _ = NewNodeFromHex("fa670379e5c2212ed93ff09769622f81f98a91e1ec8fb114d607dd25220b9088")
	expectedProof[2], _ = NewNodeFromHex("ba94ffe7edabf26ef12736f8eb5ce74d15bedb6af61444ae2906e926b1a95084")

	proof := tree.Proof()
	r.EqualValues(expectedProof, proof)

	/***************************************************
	|                       89a0                       |
	|          .ba94.                   633b           |
	|     cb59        0094        bd50       .fa67.    |
	|  0000  0100  0200  0300 =0400=.0500. 0600  0700  |
	***************************************************/
}

func TestNewProvingTreeMultiProof(t *testing.T) {
	r := require.New(t)
	tree := NewProvingTree([]uint64{1, 4})
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("89a0f1577268cc19b0a39c7a69f804fd140640c699585eb635ebb03c06154cce")
	root := tree.Root()
	r.Equal(expectedRoot, root)

	expectedProof := make([][]byte, 4)
	expectedProof[0], _ = NewNodeFromHex("0000000000000000000000000000000000000000000000000000000000000000")
	expectedProof[1], _ = NewNodeFromHex("0094579cfc7b716038d416a311465309bea202baa922b224a7b08f01599642fb")
	expectedProof[2], _ = NewNodeFromHex("0500000000000000000000000000000000000000000000000000000000000000")
	expectedProof[3], _ = NewNodeFromHex("fa670379e5c2212ed93ff09769622f81f98a91e1ec8fb114d607dd25220b9088")

	proof := tree.Proof()
	r.EqualValues(expectedProof, proof)

	/***************************************************
	|                       89a0                       |
	|           ba94                    633b           |
	|     cb59       .0094.       bd50       .fa67.    |
	| .0000.=0100= 0200  0300 =0400=.0500. 0600  0700  |
	***************************************************/
}

func TestNewProvingTreeMultiProof2(t *testing.T) {
	r := require.New(t)
	tree := NewProvingTree([]uint64{0, 1, 4})
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("89a0f1577268cc19b0a39c7a69f804fd140640c699585eb635ebb03c06154cce")
	root := tree.Root()
	r.Equal(expectedRoot, root)

	expectedProof := make([][]byte, 3)
	expectedProof[0], _ = NewNodeFromHex("0094579cfc7b716038d416a311465309bea202baa922b224a7b08f01599642fb")
	expectedProof[1], _ = NewNodeFromHex("0500000000000000000000000000000000000000000000000000000000000000")
	expectedProof[2], _ = NewNodeFromHex("fa670379e5c2212ed93ff09769622f81f98a91e1ec8fb114d607dd25220b9088")

	proof := tree.Proof()
	r.EqualValues(expectedProof, proof)

	/***************************************************
	|                       89a0                       |
	|           ba94                    633b           |
	|     cb59       .0094.       bd50       .fa67.    |
	| =0000==0100= 0200  0300 =0400=.0500. 0600  0700  |
	***************************************************/
}

func NewNodeFromUint64(i uint64) []byte {
	b := make([]byte, NodeSize)
	binary.LittleEndian.PutUint64(b, i)
	return b
}

func NewNodeFromHex(s string) ([]byte, error) {
	return hex.DecodeString(s)
}

// Caching tests:

func TestNewCachingTree(t *testing.T) {
	r := require.New(t)
	treeCache := cache.NewCacheWithLayerFactories(
		[]cache.LayerFactory{cache.MakeMemoryReadWriterFactory(0)},
	)
	tree := NewCachingTree(treeCache)
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("89a0f1577268cc19b0a39c7a69f804fd140640c699585eb635ebb03c06154cce")
	root := tree.Root()
	r.Equal(expectedRoot, root)

	r.Equal(uint64(8), treeCache.GetLayerReader(0).Width())
	r.Equal(uint64(4), treeCache.GetLayerReader(1).Width())
	r.Equal(uint64(2), treeCache.GetLayerReader(2).Width())
	r.Equal(uint64(1), treeCache.GetLayerReader(3).Width())
	cacheRoot, err := treeCache.GetLayerReader(3).ReadNext()
	r.NoError(err)
	r.Equal(cacheRoot, expectedRoot)

	//treeCache.Print(0 , 3)
}

func BenchmarkNewCachingTreeSmall(b *testing.B) {
	var size uint64 = 1 << 23
	treeCache := cache.NewCacheWithLayerFactories(
		[]cache.LayerFactory{cache.MakeMemoryReadWriterFactory(7)},
	)
	start := time.Now()
	tree := NewCachingTree(treeCache)
	for i := uint64(0); i < size; i++ {
		_ = tree.AddLeaf(NewNodeFromUint64(i))
	}
	b.Log(time.Since(start))
	/*
	   merkle_test.go:242: 3.054842184s
	*/
}

func TestSparseBoolStack(t *testing.T) {
	r := require.New(t)

	allFalse := newSparseBoolStack([]uint64{})
	for i := 0; i < 1000; i++ {
		r.False(allFalse.Pop())
	}

	allTrue := newSparseBoolStack([]uint64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
	for i := 0; i < 10; i++ {
		r.True(allTrue.Pop())
	}

	rounds := make([]uint64, 0, 100)
	for i := 0; i < 1000; i += 10 {
		rounds = append(rounds, uint64(i))
	}
	roundsTrue := newSparseBoolStack(rounds)
	for i := 0; i < 1000; i++ {
		if i%10 == 0 {
			r.True(roundsTrue.Pop())
		} else {
			r.False(roundsTrue.Pop())
		}
	}
}

func TestEmptyNode(t *testing.T) {
	r := require.New(t)

	r.True(emptyNode.IsEmpty())
	r.False(emptyNode.onProvenPath)
}
