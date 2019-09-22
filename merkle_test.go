package merkle_test

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/spacemeshos/merkle-tree"
	"github.com/spacemeshos/merkle-tree/cache"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

var (
	NewTree                                 = merkle.NewTree
	NewTreeBuilder                          = merkle.NewTreeBuilder
	NewProvingTree                          = merkle.NewProvingTree
	NewCachingTree                          = merkle.NewCachingTree
	GenerateProof                           = merkle.GenerateProof
	ValidatePartialTree                     = merkle.ValidatePartialTree
	ValidatePartialTreeWithParkingSnapshots = merkle.ValidatePartialTreeWithParkingSnapshots
	GetSha256Parent                         = merkle.GetSha256Parent
	GetNode                                 = merkle.GetNode
	setOf                                   = merkle.SetOf
	newSparseBoolStack                      = merkle.NewSparseBoolStack
	emptyNode                               = merkle.EmptyNode
	NodeSize                                = merkle.NodeSize
)

type (
	set          = merkle.Set
	position     = merkle.Position
	validator    = merkle.Validator
	leafIterator = merkle.LeafIterator
	CacheReader  = cache.CacheReader
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
	tree, err := NewTree()
	r.NoError(err)
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
	tree, err := NewTreeBuilder().WithHashFunc(concatLeaves).WithMinHeight(3).Build()
	r.NoError(err)
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
	tree, err := NewTreeBuilder().WithHashFunc(concatLeaves).WithMinHeight(4).Build()
	r.NoError(err)
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
	tree, err := NewTreeBuilder().WithHashFunc(concatLeaves).WithMinHeight(5).Build()
	r.NoError(err)
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
	tree, err := NewTree()
	r.NoError(err)
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
	tree, err := NewTree()
	r.NoError(err)
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
	tree, err := NewTree()
	r.NoError(err)
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

	leavesToProve := setOf(0, 4, 7)

	cacheWriter := cache.NewWriter(cache.MinHeightPolicy(0), cache.MakeSliceReadWriterFactory())

	tree, err := NewTreeBuilder().
		WithLeavesToProve(leavesToProve).
		WithCacheWriter(cacheWriter).
		Build()
	r.NoError(err)
	for i := uint64(0); i < 10; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("59f32a43534fe4c4c0966421aef624267cdf65bd11f74998c60f27c7caccb12d")
	root := tree.Root()
	r.Equal(expectedRoot, root)

	cacheReader, err := cacheWriter.GetReader()
	r.NoError(err)

	assertWidth(r, 10, cacheReader.GetLayerReader(0))
	assertWidth(r, 5, cacheReader.GetLayerReader(1))
	assertWidth(r, 2, cacheReader.GetLayerReader(2))
	assertWidth(r, 1, cacheReader.GetLayerReader(3))

	cacheRoot, err := cacheReader.GetLayerReader(3).ReadNext()
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

func assertWidth(r *require.Assertions, expectedWidth int, layerReader cache.LayerReader) {
	r.NotNil(layerReader)
	width, err := layerReader.Width()
	r.NoError(err)
	r.Equal(uint64(expectedWidth), width)
}

func BenchmarkNewTree(b *testing.B) {
	var size uint64 = 1 << 28
	tree, _ := NewTree()
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
	tree, _ := NewTree()
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
	tree, _ := NewTree()
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
	tree, err := NewProvingTree(setOf(4))
	r.NoError(err)
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
	tree, err := NewProvingTree(setOf(1, 4))
	r.NoError(err)
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
	tree, err := NewProvingTree(setOf(0, 1, 4))
	r.NoError(err)
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
	cacheWriter := cache.NewWriter(cache.MinHeightPolicy(0), cache.MakeSliceReadWriterFactory())
	tree, err := NewCachingTree(cacheWriter)
	r.NoError(err)
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

	//cacheWriter.Print(0 , 3)
}

func BenchmarkNewCachingTreeSmall(b *testing.B) {
	var size uint64 = 1 << 23
	cacheWriter := cache.NewWriter(cache.MinHeightPolicy(7), cache.MakeSliceReadWriterFactory())
	start := time.Now()
	tree, _ := NewCachingTree(cacheWriter)
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

	allFalse := newSparseBoolStack(make(set))
	for i := 0; i < 1000; i++ {
		r.False(allFalse.Pop())
	}

	allTrue := newSparseBoolStack(setOf(0, 1, 2, 3, 4, 5, 6, 7, 8, 9))
	for i := 0; i < 10; i++ {
		r.True(allTrue.Pop())
	}

	rounds := make(set)
	for i := uint64(0); i < 1000; i += 10 {
		rounds[i] = true
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
	r.False(emptyNode.OnProvenPath)
}

func TestTree_GetParkedNodes(t *testing.T) {
	r := require.New(t)

	tree, err := NewTreeBuilder().Build()
	r.NoError(err)

	r.NoError(tree.AddLeaf([]byte{0}))
	r.EqualValues(
		[][]byte{{0}},
		tree.GetParkedNodes())

	r.NoError(tree.AddLeaf([]byte{1}))
	r.EqualValues(
		[][]byte{nil, decode(r, "b413f47d13ee2fe6c845b2ee141af81de858df4ec549a58b7970bb96645bc8d2")},
		tree.GetParkedNodes())

	r.NoError(tree.AddLeaf([]byte{2}))
	r.EqualValues(
		[][]byte{{2}, decode(r, "b413f47d13ee2fe6c845b2ee141af81de858df4ec549a58b7970bb96645bc8d2")},
		tree.GetParkedNodes())

	r.NoError(tree.AddLeaf([]byte{3}))
	r.EqualValues(
		[][]byte{nil, nil, decode(r, "7699a4fdd6b8b6908a344f73b8f05c8e1400f7253f544602c442ff5c65504b24")},
		tree.GetParkedNodes())
}

func TestTree_SetParkedNodes(t *testing.T) {
	r := require.New(t)

	tree, err := NewTreeBuilder().Build()
	r.NoError(err)
	r.NoError(tree.SetParkedNodes([][]byte{{0}}))
	r.NoError(tree.AddLeaf([]byte{1}))
	parkedNodes := [][]byte{nil, decode(r, "b413f47d13ee2fe6c845b2ee141af81de858df4ec549a58b7970bb96645bc8d2")}
	r.EqualValues(parkedNodes, tree.GetParkedNodes())

	tree, err = NewTreeBuilder().Build()
	r.NoError(err)
	r.NoError(tree.SetParkedNodes(parkedNodes))
	r.NoError(tree.AddLeaf([]byte{2}))
	parkedNodes = [][]byte{{2}, decode(r, "b413f47d13ee2fe6c845b2ee141af81de858df4ec549a58b7970bb96645bc8d2")}
	r.EqualValues(parkedNodes, tree.GetParkedNodes())

	tree, err = NewTreeBuilder().Build()
	r.NoError(err)
	r.NoError(tree.SetParkedNodes(parkedNodes))
	r.NoError(tree.AddLeaf([]byte{3}))
	parkedNodes = [][]byte{nil, nil, decode(r, "7699a4fdd6b8b6908a344f73b8f05c8e1400f7253f544602c442ff5c65504b24")}
	r.EqualValues(parkedNodes, tree.GetParkedNodes())
}

func decode(r *require.Assertions, hexString string) []byte {
	hash, err := hex.DecodeString(hexString)
	r.NoError(err)
	return hash
}

// Annotated example explaining how to use this package
func ExampleTree() {
	// First, we create a cache writer with caching policy and layer read-writer factory:
	cacheWriter := cache.NewWriter(cache.MinHeightPolicy(0), cache.MakeSliceReadWriterFactory())

	// We then initialize the tree:
	tree, err := NewTreeBuilder().WithCacheWriter(cacheWriter).Build()
	if err != nil {
		fmt.Println("Error while building the tree:", err.Error())
		return
	}

	// We add the leaves one-by-one:
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		if err != nil {
			fmt.Println("Error while adding a leaf:", err.Error())
			return
		}
	}

	// After adding some leaves we can access the root of the tree:
	fmt.Println(tree.Root()) // 89a0f1577268cc19b0a39c7a69f804fd140640c699585eb635ebb03c06154cce

	// If we need to generate a proof, we could derive the proven leaves from the root. Here we create a static set:
	leavesToProve := setOf(0, 4, 7)

	// We get a cache reader from the cache writer:
	cacheReader, err := cacheWriter.GetReader()
	if err != nil {
		fmt.Println("Error while getting cache reader:", err.Error())
		return
	}

	// We pass the cache into GenerateProof along with the set of leaves to prove:
	sortedProvenLeafIndices, provenLeaves, proof, err := GenerateProof(leavesToProve, cacheReader)
	if err != nil {
		fmt.Println("Error while getting generating proof:", err.Error())
		return
	}

	// We now have access to a sorted list of proven leaves, the values of those leaves and the Merkle proof for them:
	fmt.Println(sortedProvenLeafIndices) // 0 4 7
	fmt.Println(nodes(provenLeaves))     // 0000 0400 0700
	fmt.Println(nodes(proof))            // 0100 0094 0500 0600

	// We can validate these values using ValidatePartialTree:
	valid, err := ValidatePartialTree(sortedProvenLeafIndices, provenLeaves, proof, tree.Root(), GetSha256Parent)
	if err != nil {
		fmt.Println("Error while validating proof:", err.Error())
		return
	}
	fmt.Println(valid) // true

	/***************************************************
	|                       89a0                       |
	|           ba94                    633b           |
	|     cb59       .0094.       bd50        fa67     |
	| =0000=.0100. 0200  0300 =0400=.0500..0600.=0700= |
	***************************************************/
}
