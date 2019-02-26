package simpler

import (
	"encoding/binary"
	"encoding/hex"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

/*

	8-leaf tree (1st byte of each node):

	+----------------------------------+
	|                4a                |
	|        13              6c        |
	|    9d      fe      3d      6b    |
	|  00  01  02  03  04  05  06  07  |
	+----------------------------------+

*/

func TestNewTree(t *testing.T) {
	tree := NewTree(GetSha256Parent)
	for i := uint64(0); i < 8; i++ {
		tree.AddLeaf(NewNodeFromUint64(i))
	}
	expectedRoot, _ := NewNodeFromHex("4a2ca61d1fd537170785a8575d424634713c82e7392e67795a807653e498cfd0")
	root, err := tree.Root()
	require.NoError(t, err)
	require.Equal(t, expectedRoot, root)
}

func TestNewTreeNotPowerOf2(t *testing.T) {
	tree := NewTree(GetSha256Parent)
	for i := uint64(0); i < 9; i++ {
		tree.AddLeaf(NewNodeFromUint64(i))
	}
	root, err := tree.Root()
	require.Error(t, err)
	require.Nil(t, root)
}

func BenchmarkNewTree(b *testing.B) {
	var size uint64 = 1 << 28
	tree := NewTree(GetSha256Parent)
	for i := uint64(0); i < size; i++ {
		tree.AddLeaf(NewNodeFromUint64(i))
	}
	/*
		goos: darwin
		goarch: amd64
		pkg: github.com/spacemeshos/merkle-tree
		BenchmarkNewTree-8   	       1	91585669075 ns/op (original implementation)
		BenchmarkNewTree-8   	       1	105642465926 ns/op (initial new implementation)
		BenchmarkNewTree-8   	       1	98904481383 ns/op (current implementation)
		PASS
	*/
}

func BenchmarkNewTreeSmall(b *testing.B) {
	var size uint64 = 1 << 23
	start := time.Now()
	tree := NewTree(GetSha256Parent)
	for i := uint64(0); i < size; i++ {
		tree.AddLeaf(NewNodeFromUint64(i))
	}
	b.Log(time.Since(start))
}

func BenchmarkNewTreeNoHashing(b *testing.B) {
	var size uint64 = 1 << 28
	tree := NewTree(func(leftChild, rightChild []byte) []byte {
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
		BenchmarkNewTreeNoHashing-8   	       1	13525018261 ns/op
		BenchmarkNewTreeNoHashing-8   	       1	23559855346 ns/op
		BenchmarkNewTreeNoHashing-8   	       1	15234160295 ns/op
		PASS
	*/
}

/*
	28 layer tree takes 91.5 seconds to construct. Overhead (no hashing) is 13.5 seconds. Net: 78 seconds.
	(8.5GB @ 32b leaves) => x30 256GB => 39 minutes for hashing, 7 minutes overhead.

	Reading 256GB from a magnetic disk should take ~30 minutes.
*/

func TestNewProvingTree(t *testing.T) {
	tree := NewProvingTree(GetSha256Parent, []uint64{4, 8})
	for i := uint64(0); i < 8; i++ {
		tree.AddLeaf(NewNodeFromUint64(i))
	}
	expectedRoot, _ := NewNodeFromHex("4a2ca61d1fd537170785a8575d424634713c82e7392e67795a807653e498cfd0")
	root, err := tree.Root()
	require.NoError(t, err)
	require.Equal(t, expectedRoot, root)

	expectedProof := make([][]byte, 3)
	expectedProof[0], _ = NewNodeFromHex("0500000000000000")
	expectedProof[1], _ = NewNodeFromHex("6b2e10cb2111114ce942174c38e7ea38864cc364a8fe95c66869c85888d812da")
	expectedProof[2], _ = NewNodeFromHex("13c04a6157aa640f711d230a4f04bc2b19e75df1127dfc899f025f3aa282912d")

	proof, err := tree.Proof()
	require.NoError(t, err)
	require.EqualValues(t, expectedProof, proof)

	/***********************************
	|                4a                |
	|       .13.             6c        |
	|    9d      fe      3d     .6b.   |
	|  00  01  02  03 =04=.05. 06  07  |
	***********************************/
}

func TestNewProvingTreeMultiProof(t *testing.T) {
	tree := NewProvingTree(GetSha256Parent, []uint64{1, 4})
	for i := uint64(0); i < 8; i++ {
		tree.AddLeaf(NewNodeFromUint64(i))
	}
	expectedRoot, _ := NewNodeFromHex("4a2ca61d1fd537170785a8575d424634713c82e7392e67795a807653e498cfd0")
	root, err := tree.Root()
	require.NoError(t, err)
	require.Equal(t, expectedRoot, root)

	expectedProof := make([][]byte, 4)
	expectedProof[0], _ = NewNodeFromHex("0000000000000000")
	expectedProof[1], _ = NewNodeFromHex("fe6d3d3bb5dd778af1128cc7b2b33668d51b9a52dfc8f2342be37ddc06a0072d")
	expectedProof[2], _ = NewNodeFromHex("0500000000000000")
	expectedProof[3], _ = NewNodeFromHex("6b2e10cb2111114ce942174c38e7ea38864cc364a8fe95c66869c85888d812da")

	proof, err := tree.Proof()
	require.NoError(t, err)
	require.EqualValues(t, expectedProof, proof)

	/***********************************
	|                4a                |
	|        13              6c        |
	|    9d     .fe.     3d     .6b.   |
	| .00.=01= 02  03 =04=.05. 06  07  |
	***********************************/
}

func TestNewProvingTreeMultiProof2(t *testing.T) {
	tree := NewProvingTree(GetSha256Parent, []uint64{0, 1, 4})
	for i := uint64(0); i < 8; i++ {
		tree.AddLeaf(NewNodeFromUint64(i))
	}
	expectedRoot, _ := NewNodeFromHex("4a2ca61d1fd537170785a8575d424634713c82e7392e67795a807653e498cfd0")
	root, err := tree.Root()
	require.NoError(t, err)
	require.Equal(t, expectedRoot, root)

	expectedProof := make([][]byte, 3)
	expectedProof[0], _ = NewNodeFromHex("fe6d3d3bb5dd778af1128cc7b2b33668d51b9a52dfc8f2342be37ddc06a0072d")
	expectedProof[1], _ = NewNodeFromHex("0500000000000000")
	expectedProof[2], _ = NewNodeFromHex("6b2e10cb2111114ce942174c38e7ea38864cc364a8fe95c66869c85888d812da")

	proof, err := tree.Proof()
	require.NoError(t, err)
	require.EqualValues(t, expectedProof, proof)

	/***********************************
	|                4a                |
	|        13              6c        |
	|    9d     .fe.     3d     .6b.   |
	| =00==01= 02  03 =04=.05. 06  07  |
	***********************************/
}

func NewNodeFromUint64(i uint64) []byte {
	const bytesInUint64 = 8
	b := make([]byte, bytesInUint64)
	binary.LittleEndian.PutUint64(b, i)
	return b
}

func NewNodeFromHex(s string) ([]byte, error) {
	return hex.DecodeString(s)
}
