package merkle_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/merkle-tree/cache"
)

func TestValidatePartialTree(t *testing.T) {
	req := require.New(t)

	leafIndices := []uint64{3}
	leaves := [][]byte{NewNodeFromUint64(3)}
	proof := [][]byte{
		NewNodeFromUint64(0),
		NewNodeFromUint64(0),
		NewNodeFromUint64(0),
	}
	root, _ := NewNodeFromHex("2657509b700c67b205c5196ee9a231e0fe567f1dae4a15bb52c0de813d65677a")
	valid, err := ValidatePartialTree(leafIndices, leaves, proof, root, GetSha256Parent)
	req.NoError(err)
	req.True(valid, "Proof should be valid, but isn't")
}

func TestValidatePartialTreeProofs(t *testing.T) {
	for n := 1; n <= 64; n++ {
		for l := 0; l < n; l++ {
			t.Run(fmt.Sprintf("N%d/L%d", n, l), func(t *testing.T) {
				r1, p1 := validateFromCache(t, n, l)
				r2, p2 := validateFromScratch(t, n, l)
				require.Equal(t, r1, r2)
				require.Equal(t, p1, p2)
			})
		}
	}
}

func validateFromCache(t *testing.T, n, l int) ([]byte, [][]byte) {
	req := require.New(t)
	leafIndices := []uint64{uint64(l)}
	metaFactory := cache.MakeSliceReadWriterFactory()

	treeCache := cache.NewWriter(
		cache.MinHeightPolicy(0),
		metaFactory,
	)

	tree, err := NewTreeBuilder().WithCacheWriter(treeCache).Build()
	req.NoError(err)
	for i := uint64(0); i < uint64(n); i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		req.NoError(err)
	}

	root := tree.Root()
	reader, err := treeCache.GetReader()
	require.NoError(t, err)
	_, leaves, nodes, err := GenerateProof(setOf(leafIndices...), reader)
	require.NoError(t, err)
	valid, _, err := ValidatePartialTreeWithParkingSnapshots(leafIndices, leaves, nodes, root, GetSha256Parent)
	req.NoError(err)
	req.True(valid, "Proof should be valid, but isn't")
	return root, nodes
}

func validateFromScratch(t *testing.T, n, l int) ([]byte, [][]byte) {
	req := require.New(t)

	leafIndices := []uint64{uint64(l)}

	tree, err := NewProvingTree(setOf(leafIndices...))
	req.NoError(err)
	for i := uint64(0); i < uint64(n); i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		req.NoError(err)
	}

	root, nodes := tree.RootAndProof()
	leaves := [][]byte{
		NewNodeFromUint64(uint64(l)),
	}
	valid, _, err := ValidatePartialTreeWithParkingSnapshots(leafIndices, leaves, nodes, root, GetSha256Parent)
	req.NoError(err)
	req.True(valid, "Proof should be valid, but isn't")
	return root, nodes
}

func TestValidatePartialTreeMulti2(t *testing.T) {
	req := require.New(t)

	leafIndices := []uint64{0, 1, 4}
	leaves := [][]byte{
		NewNodeFromUint64(0),
		NewNodeFromUint64(1),
		NewNodeFromUint64(4),
	}
	tree, err := NewProvingTree(setOf(leafIndices...))
	req.NoError(err)
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		req.NoError(err)
	}
	root, proof := tree.RootAndProof() // 89a0f1577268cc19b0a39c7a69f804fd140640c699585eb635ebb03c06154cce, 009 05 fa

	valid, err := ValidatePartialTree(leafIndices, leaves, proof, root, GetSha256Parent)
	req.NoError(err)
	req.True(valid, "Proof should be valid, but isn't")

	/***************************************************
	|                       89a0                       |
	|           ba94                    633b           |
	|     cb59       .0094.       bd50       .fa67.    |
	| =0000==0100= 0200  0300 =0400=.0500. 0600  0700  |
	***************************************************/
}

func TestValidatePartialTreeParkingSnapshots(t *testing.T) {
	req := require.New(t)

	leafIndices := []uint64{4, 6}
	leaves := [][]byte{
		NewNodeFromUint64(4),
		NewNodeFromUint64(6),
	}
	tree, err := NewProvingTree(setOf(leafIndices...))
	req.NoError(err)
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		req.NoError(err)
	}
	root, proof := tree.RootAndProof() // 89a0f1577268cc19b0a39c7a69f804fd140640c699585eb635ebb03c06154cce, 05 07 ba

	valid, parkingSnapshots, err := ValidatePartialTreeWithParkingSnapshots(leafIndices, leaves, proof, root, GetSha256Parent)
	req.NoError(err)
	req.True(valid, "Proof should be valid, but isn't")
	req.Equal(
		"[[  ba94ffe7edabf26ef12736f8eb5ce74d15bedb6af61444ae2906e926b1a95084] "+
			"[ bd50456d5ad175ae99a1612a53ca229124b65d3eaabd9ff9c7ab979a385cf6b3 ba94ffe7edabf26ef12736f8eb5ce74d15bedb6af61444ae2906e926b1a95084]]",
		fmt.Sprintf("%x", parkingSnapshots))

	/***************************************************
	|                       89a0                       |
	|          .ba94.                   633b           |
	|     cb59        0094        bd50        fa67     |
	|  0000  0100  0200  0300 =0400=.0500.=0600=.0700. |
	***************************************************/
}

func TestValidatePartialTreeMultiUnbalanced(t *testing.T) {
	req := require.New(t)

	leafIndices := []uint64{0, 4, 7}
	leaves := [][]byte{
		NewNodeFromUint64(0),
		NewNodeFromUint64(4),
		NewNodeFromUint64(7),
	}
	tree, err := NewProvingTree(setOf(leafIndices...))
	req.NoError(err)
	for i := uint64(0); i < 10; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		req.NoError(err)
	}
	var proof nodes
	root, proof := tree.RootAndProof()
	// 59f32a43534fe4c4c0966421aef624267cdf65bd11f74998c60f27c7caccb12d, 0100 0094 0500 0600 bc68

	valid, err := ValidatePartialTree(leafIndices, leaves, proof, root, GetSha256Parent)
	req.NoError(err)
	req.True(valid, "Proof should be valid, but isn't")

	/***************************************************************
	|                       89a0                                   |
	|           ba94                    633b                       |
	|     cb59       .0094.       bd50        fa67       .baf8.    |
	| =0000=.0100. 0200  0300 =0400=.0500..0600.=0700= 0800  0900  |
	***************************************************************/
}

func TestValidatePartialTreeMultiUnbalanced2(t *testing.T) {
	req := require.New(t)

	leafIndices := []uint64{0, 4, 7, 9}
	leaves := [][]byte{
		NewNodeFromUint64(0),
		NewNodeFromUint64(4),
		NewNodeFromUint64(7),
		NewNodeFromUint64(9),
	}
	tree, err := NewProvingTree(setOf(leafIndices...))
	req.NoError(err)
	for i := uint64(0); i < 10; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		req.NoError(err)
	}
	var proof nodes
	root, proof := tree.RootAndProof()
	// 59f32a43534fe4c4c0966421aef624267cdf65bd11f74998c60f27c7caccb12d, 0100 0094 0500 0600 0800 0000 0000

	valid, err := ValidatePartialTree(leafIndices, leaves, proof, root, GetSha256Parent)
	req.NoError(err)
	req.True(valid, "Proof should be valid, but isn't")

	/***************************************************************
	|                       89a0                                   |
	|           ba94                    633b                       |
	|     cb59       .0094.       bd50        fa67        baf8     |
	| =0000=.0100. 0200  0300 =0400=.0500..0600.=0700=.0800.=0900= |
	***************************************************************/
}

func TestValidatePartialTreeUnbalanced(t *testing.T) {
	req := require.New(t)

	leafIndices := []uint64{9}
	leaves := [][]byte{
		NewNodeFromUint64(9),
	}
	tree, err := NewProvingTree(setOf(leafIndices...))
	req.NoError(err)
	for i := uint64(0); i < 10; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		req.NoError(err)
	}
	var proof nodes
	root, proof := tree.RootAndProof()
	// 59f32a43534fe4c4c0966421aef624267cdf65bd11f74998c60f27c7caccb12d, 0800 0000 0000 89a0

	valid, err := ValidatePartialTree(leafIndices, leaves, proof, root, GetSha256Parent)
	req.NoError(err)
	req.True(valid, "Proof should be valid, but isn't")

	/***************************************************************
	|                      .89a0.                                  |
	|           ba94                    633b                       |
	|     cb59        0094        bd50        fa67        baf8     |
	|  0000  0100  0200  0300  0400  0500  0600  0700 .0800.=0900= |
	***************************************************************/
}

func BenchmarkValidatePartialTree(b *testing.B) {
	req := require.New(b)

	leafIndices := []uint64{100, 1000, 10000, 100000, 1000000, 2000000, 4000000, 8000000}
	var leaves [][]byte
	for _, i := range leafIndices {
		leaves = append(leaves, NewNodeFromUint64(i))
	}
	tree, err := NewProvingTree(setOf(leafIndices...))
	req.NoError(err)
	for i := uint64(0); i < 1<<23; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		req.NoError(err)
	}
	root, proof := tree.RootAndProof()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		valid, err := ValidatePartialTree(leafIndices, leaves, proof, root, GetSha256Parent)
		// valid, _, err := ValidatePartialTreeWithParkingSnapshots(leafIndices, leaves, proof, root, GetSha256Parent)
		req.NoError(err)
		req.True(valid, "Proof should be valid, but isn't")
	}

	/*
		BenchmarkValidatePartialTree-8   	   20000	     63520 ns/op
		BenchmarkValidatePartialTree-8   	   20000	     73310 ns/op +15% due to collecting parking snapshots
	*/

	/***************************************************
	|                       89a0                       |
	|           ba94                    633b           |
	|     cb59       .0094.       bd50       .fa67.    |
	| =0000==0100= 0200  0300 =0400=.0500. 0600  0700  |
	***************************************************/
}

func TestValidatePartialTreeErrors(t *testing.T) {
	req := require.New(t)

	leafIndices := []uint64{3, 5}
	leaves := [][]byte{NewNodeFromUint64(3)}
	proof := [][]byte{
		NewNodeFromUint64(0),
		NewNodeFromUint64(0),
		NewNodeFromUint64(0),
	}
	root, _ := NewNodeFromHex("2657509b700c67b205c5196ee9a231e0fe567f1dae4a15bb52c0de813d65677a")
	valid, err := ValidatePartialTree(leafIndices, leaves, proof, root, GetSha256Parent)
	req.EqualError(err, "number of leaves (1) must equal number of indices (2)")
	req.False(valid)

	valid, err = ValidatePartialTree([]uint64{}, [][]byte{}, proof, root, GetSha256Parent)
	req.EqualError(err, "at least one leaf is required for validation")
	req.False(valid)

	leafIndices = []uint64{5, 3}
	leaves = [][]byte{NewNodeFromUint64(5), NewNodeFromUint64(3)}
	valid, err = ValidatePartialTree(leafIndices, leaves, proof, root, GetSha256Parent)
	req.EqualError(err, "leafIndices are not sorted")
	req.False(valid)

	leafIndices = []uint64{3, 3}
	leaves = [][]byte{NewNodeFromUint64(5), NewNodeFromUint64(3)}
	valid, err = ValidatePartialTree(leafIndices, leaves, proof, root, GetSha256Parent)
	req.EqualError(err, "leafIndices contain duplicates")
	req.False(valid)
}

func TestValidator_calcRoot(t *testing.T) {
	r := require.New(t)
	v := validator{
		Leaves:         &leafIterator{},
		ProofNodes:     nil,
		Hash:           nil,
		StoreSnapshots: false,
	}

	root, _, err := v.CalcRoot(0)

	r.EqualError(err, "no more items")
	r.Nil(root)
}
