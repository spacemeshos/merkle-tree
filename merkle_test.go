package merkle

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
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

func TestNewTree(t *testing.T) {
	r := require.New(t)
	tree := NewTree(GetSha256Parent)
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("89a0f1577268cc19b0a39c7a69f804fd140640c699585eb635ebb03c06154cce")
	root, err := tree.Root()
	r.NoError(err)
	r.Equal(expectedRoot, root)
}

func TestNewTreeNotPowerOf2(t *testing.T) {
	r := require.New(t)
	tree := NewTree(GetSha256Parent)
	for i := uint64(0); i < 9; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	root, err := tree.Root()
	r.Error(err)
	r.Nil(root)
}

func BenchmarkNewTree(b *testing.B) {
	var size uint64 = 1 << 28
	tree := NewTree(GetSha256Parent)
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
	tree := NewTree(GetSha256Parent)
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
	tree := NewTree(func(leftChild, rightChild []byte) []byte {
		arr := [32]byte{}
		return arr[:]
	})
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
	tree := NewProvingTree(GetSha256Parent, []uint64{4})
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("89a0f1577268cc19b0a39c7a69f804fd140640c699585eb635ebb03c06154cce")
	root, err := tree.Root()
	r.NoError(err)
	r.Equal(expectedRoot, root)

	expectedProof := make([][]byte, 3)
	expectedProof[0], _ = NewNodeFromHex("0500000000000000000000000000000000000000000000000000000000000000")
	expectedProof[1], _ = NewNodeFromHex("fa670379e5c2212ed93ff09769622f81f98a91e1ec8fb114d607dd25220b9088")
	expectedProof[2], _ = NewNodeFromHex("ba94ffe7edabf26ef12736f8eb5ce74d15bedb6af61444ae2906e926b1a95084")

	proof, err := tree.Proof()
	r.NoError(err)
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
	tree := NewProvingTree(GetSha256Parent, []uint64{1, 4})
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("89a0f1577268cc19b0a39c7a69f804fd140640c699585eb635ebb03c06154cce")
	root, err := tree.Root()
	r.NoError(err)
	r.Equal(expectedRoot, root)

	expectedProof := make([][]byte, 4)
	expectedProof[0], _ = NewNodeFromHex("0000000000000000000000000000000000000000000000000000000000000000")
	expectedProof[1], _ = NewNodeFromHex("0094579cfc7b716038d416a311465309bea202baa922b224a7b08f01599642fb")
	expectedProof[2], _ = NewNodeFromHex("0500000000000000000000000000000000000000000000000000000000000000")
	expectedProof[3], _ = NewNodeFromHex("fa670379e5c2212ed93ff09769622f81f98a91e1ec8fb114d607dd25220b9088")

	proof, err := tree.Proof()
	r.NoError(err)
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
	tree := NewProvingTree(GetSha256Parent, []uint64{0, 1, 4})
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("89a0f1577268cc19b0a39c7a69f804fd140640c699585eb635ebb03c06154cce")
	root, err := tree.Root()
	r.NoError(err)
	r.Equal(expectedRoot, root)

	expectedProof := make([][]byte, 3)
	expectedProof[0], _ = NewNodeFromHex("0094579cfc7b716038d416a311465309bea202baa922b224a7b08f01599642fb")
	expectedProof[1], _ = NewNodeFromHex("0500000000000000000000000000000000000000000000000000000000000000")
	expectedProof[2], _ = NewNodeFromHex("fa670379e5c2212ed93ff09769622f81f98a91e1ec8fb114d607dd25220b9088")

	proof, err := tree.Proof()
	r.NoError(err)
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

type sliceReadWriter struct {
	slice    [][]byte
	position uint64
}

func (s *sliceReadWriter) Width() uint64 {
	return uint64(len(s.slice))
}

func (s *sliceReadWriter) Seek(index uint64) error {
	if index >= uint64(len(s.slice)) {
		return io.EOF
	}
	s.position = index
	return nil
}

func (s *sliceReadWriter) ReadNext() ([]byte, error) {
	if s.position >= uint64(len(s.slice)) {
		return nil, io.EOF
	}
	value := make([]byte, NodeSize)
	copy(value, s.slice[s.position])
	s.position++
	return value, nil
}

func (s *sliceReadWriter) Write(p []byte) (n int, err error) {
	s.slice = append(s.slice, p)
	return len(p), nil
}

func TestNewCachingTree(t *testing.T) {
	r := require.New(t)
	sliceWriters := make(map[uint]*sliceReadWriter)
	for i := uint(0); i < 4; i++ {
		sliceWriters[i] = &sliceReadWriter{}
	}
	tree := NewCachingTree(GetSha256Parent, WritersFromSliceReadWriters(sliceWriters))
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("89a0f1577268cc19b0a39c7a69f804fd140640c699585eb635ebb03c06154cce")
	root, err := tree.Root()
	r.NoError(err)
	r.Equal(expectedRoot, root)

	r.Len(sliceWriters[0].slice, 8)
	r.Len(sliceWriters[1].slice, 4)
	r.Len(sliceWriters[2].slice, 2)
	r.Len(sliceWriters[3].slice, 1)
	r.Equal(sliceWriters[3].slice[0], expectedRoot)

	// printCache(0, 3, sliceWriters)
}

func printCache(bottom, top int, sliceWriters map[uint]*sliceReadWriter) {
	for i := top; i >= bottom; i-- {
		print("| ")
		for _, n := range sliceWriters[uint(i)].slice {
			printSpaces(numSpaces(i))
			fmt.Print(hex.EncodeToString(n[:2]))
			printSpaces(numSpaces(i))
		}
		println(" |")
	}
}

func numSpaces(n int) int {
	res := 1
	for i := 0; i < n; i++ {
		res += 3 * (1 << uint(i))
	}
	return res
}

func printSpaces(n int) {
	for i := 0; i < n; i++ {
		print(" ")
	}
}

func BenchmarkNewCachingTreeSmall(b *testing.B) {
	var size uint64 = 1 << 23
	cache := make(map[uint]io.Writer)
	for i := uint(7); i < 23; i++ {
		cache[i] = &sliceReadWriter{}
	}
	start := time.Now()
	tree := NewCachingTree(GetSha256Parent, cache)
	for i := uint64(0); i < size; i++ {
		_ = tree.AddLeaf(NewNodeFromUint64(i))
	}
	b.Log(time.Since(start))
	/*
	   merkle_test.go:242: 3.054842184s
	*/
}
