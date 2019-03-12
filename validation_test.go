package merkle

import (
	"github.com/stretchr/testify/require"
	"testing"
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

func TestValidatePartialTreeForRealz(t *testing.T) {
	req := require.New(t)

	leafIndices := []uint64{4}
	leaves := [][]byte{NewNodeFromUint64(4)}
	tree := NewProvingTree(GetSha256Parent, leafIndices)
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		req.NoError(err)
	}
	root, err := tree.Root() // 89a0f1577268cc19b0a39c7a69f804fd140640c699585eb635ebb03c06154cce
	req.NoError(err)
	proof, err := tree.Proof() // 05 fa ba
	req.NoError(err)

	valid, err := ValidatePartialTree(leafIndices, leaves, proof, root, GetSha256Parent)
	req.NoError(err)
	req.True(valid, "Proof should be valid, but isn't")

	/***************************************************
	|                       89a0                       |
	|          .ba94.                   633b           |
	|     cb59        0094        bd50       .fa67.    |
	|  0000  0100  0200  0300 =0400=.0500. 0600  0700  |
	***************************************************/
}

func TestValidatePartialTreeMulti(t *testing.T) {
	req := require.New(t)

	leafIndices := []uint64{1, 4}
	leaves := [][]byte{
		NewNodeFromUint64(1),
		NewNodeFromUint64(4),
	}
	tree := NewProvingTree(GetSha256Parent, leafIndices)
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		req.NoError(err)
	}
	root, err := tree.Root() // 89a0f1577268cc19b0a39c7a69f804fd140640c699585eb635ebb03c06154cce
	req.NoError(err)
	proof, err := tree.Proof() // 05 fa ba
	req.NoError(err)

	valid, err := ValidatePartialTree(leafIndices, leaves, proof, root, GetSha256Parent)
	req.NoError(err)
	req.True(valid, "Proof should be valid, but isn't")

	/***************************************************
	|                       89a0                       |
	|           ba94                    633b           |
	|     cb59       .0094.       bd50       .fa67.    |
	| .0000.=0100= 0200  0300 =0400=.0500. 0600  0700  |
	***************************************************/
}

func TestValidatePartialTreeMulti2(t *testing.T) {
	req := require.New(t)

	leafIndices := []uint64{0, 1, 4}
	leaves := [][]byte{
		NewNodeFromUint64(0),
		NewNodeFromUint64(1),
		NewNodeFromUint64(4),
	}
	tree := NewProvingTree(GetSha256Parent, leafIndices)
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		req.NoError(err)
	}
	root, err := tree.Root() // 89a0f1577268cc19b0a39c7a69f804fd140640c699585eb635ebb03c06154cce
	req.NoError(err)
	proof, err := tree.Proof() // 05 fa ba
	req.NoError(err)

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

func BenchmarkValidatePartialTree(b *testing.B) {
	req := require.New(b)

	leafIndices := []uint64{100, 1000, 10000, 100000, 1000000, 2000000, 4000000, 8000000}
	var leaves [][]byte
	for _, i := range leafIndices {
		leaves = append(leaves, NewNodeFromUint64(i))
	}
	tree := NewProvingTree(GetSha256Parent, leafIndices)
	for i := uint64(0); i < 1<<23; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		req.NoError(err)
	}
	root, err := tree.Root()
	req.NoError(err)
	proof, err := tree.Proof()
	req.NoError(err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		valid, err := ValidatePartialTree(leafIndices, leaves, proof, root, GetSha256Parent)
		req.NoError(err)
		req.True(valid, "Proof should be valid, but isn't")
	}

	/***************************************************
	|                       89a0                       |
	|           ba94                    633b           |
	|     cb59       .0094.       bd50       .fa67.    |
	| =0000==0100= 0200  0300 =0400=.0500. 0600  0700  |
	***************************************************/
}
