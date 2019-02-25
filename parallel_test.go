package merkle

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNewParallelTree(t *testing.T) {
	tree := NewParallelTree(GetSha256Parent)
	for i := uint64(0); i < 8; i++ {
		tree.AddLeaf(NewNodeFromUint64(i))
	}
	expectedRoot, _ := NewNodeFromHex("4a2ca61d1fd537170785a8575d424634713c82e7392e67795a807653e498cfd0")
	root, err := tree.Root()
	require.NoError(t, err)
	require.Equal(t, expectedRoot, root)
}

func TestNewParallelTreeNotPowerOf2(t *testing.T) {
	tree := NewParallelTree(GetSha256Parent)
	for i := uint64(0); i < 9; i++ {
		tree.AddLeaf(NewNodeFromUint64(i))
	}
	root, err := tree.Root()
	require.Error(t, err)
	require.Nil(t, root)
}

func BenchmarkNewParallelTree(b *testing.B) {
	var size uint64 = 1 << 28
	tree := NewParallelTree(GetSha256Parent)
	for i := uint64(0); i < size; i++ {
		tree.AddLeaf(NewNodeFromUint64(i))
	}
	_, _ = tree.Root() // this waits for all threads to complete before returning
	/*
		goos: darwin
		goarch: amd64
		pkg: github.com/spacemeshos/merkle-tree
		BenchmarkNewParallelTree-8   	       1	154651620156 ns/op
		PASS
	*/
}

func BenchmarkNewParallelTreeSmall(b *testing.B) {
	var size uint64 = 1 << 23
	/*
		var size uint64 = 1 << 23

		BenchmarkNewParallelTreeSmall-8   	       1	12247472307 ns/op (1 goroutines)
		BenchmarkNewParallelTreeSmall-8   	       1	8584817082 ns/op (2 goroutines)
		BenchmarkNewParallelTreeSmall-8   	       1	6974472850 ns/op (4 goroutines)
		BenchmarkNewParallelTreeSmall-8   	       1	5494326220 ns/op (8 goroutines)
		BenchmarkNewParallelTreeSmall-8   	       1	4673070897 ns/op (16 goroutines)
		BenchmarkNewParallelTreeSmall-8   	       1	4247948069 ns/op (32 goroutines)
	*/
	start := time.Now()

	tree := NewParallelTree(GetSha256Parent)
	for i := uint64(0); i < size; i++ {
		tree.AddLeaf(NewNodeFromUint64(i))
	}
	_, _ = tree.Root() // this waits for all threads to complete before returning

	b.Log(time.Since(start))
}

func BenchmarkNewParallelTreeNoHashing(b *testing.B) {
	var size uint64 = 1 << 28
	tree := NewParallelTree(func(leftChild, rightChild Node) Node {
		arr := [32]byte{}
		return arr[:]
	})
	for i := uint64(0); i < size; i++ {
		tree.AddLeaf(NewNodeFromUint64(i))
	}
	/*
		goos: darwin
		goarch: amd64
		pkg: github.com/spacemeshos/merkle-tree
		BenchmarkNewParallelTreeNoHashing-8   	       1	154306573321 ns/op

		PASS
	*/
}
