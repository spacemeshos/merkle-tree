package merkle

import (
	"encoding/binary"
	"encoding/hex"
	"github.com/stretchr/testify/require"
	"io"
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
	r := require.New(t)
	tree := NewTree(GetSha256Parent)
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("4a2ca61d1fd537170785a8575d424634713c82e7392e67795a807653e498cfd0")
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
		BenchmarkNewTree-8   	       1	93337406540 ns/op
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
	    merkle_test.go:72: 2.920571503s
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
		PASS
	*/
}

/*
	28 layer tree takes 93 seconds to construct. Overhead (no hashing) is 14.5 seconds. Net: 78.5 seconds.
	(8.5GB @ 32b leaves) => x30 256GB => 39 minutes for hashing, 7.5 minutes overhead.

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
	expectedRoot, _ := NewNodeFromHex("4a2ca61d1fd537170785a8575d424634713c82e7392e67795a807653e498cfd0")
	root, err := tree.Root()
	r.NoError(err)
	r.Equal(expectedRoot, root)

	expectedProof := make([][]byte, 3)
	expectedProof[0], _ = NewNodeFromHex("0500000000000000")
	expectedProof[1], _ = NewNodeFromHex("6b2e10cb2111114ce942174c38e7ea38864cc364a8fe95c66869c85888d812da")
	expectedProof[2], _ = NewNodeFromHex("13c04a6157aa640f711d230a4f04bc2b19e75df1127dfc899f025f3aa282912d")

	proof, err := tree.Proof()
	r.NoError(err)
	r.EqualValues(expectedProof, proof)

	/***********************************
	|                4a                |
	|       .13.             6c        |
	|    9d      fe      3d     .6b.   |
	|  00  01  02  03 =04=.05. 06  07  |
	***********************************/
}

func TestNewProvingTreeMultiProof(t *testing.T) {
	r := require.New(t)
	tree := NewProvingTree(GetSha256Parent, []uint64{1, 4})
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("4a2ca61d1fd537170785a8575d424634713c82e7392e67795a807653e498cfd0")
	root, err := tree.Root()
	r.NoError(err)
	r.Equal(expectedRoot, root)

	expectedProof := make([][]byte, 4)
	expectedProof[0], _ = NewNodeFromHex("0000000000000000")
	expectedProof[1], _ = NewNodeFromHex("fe6d3d3bb5dd778af1128cc7b2b33668d51b9a52dfc8f2342be37ddc06a0072d")
	expectedProof[2], _ = NewNodeFromHex("0500000000000000")
	expectedProof[3], _ = NewNodeFromHex("6b2e10cb2111114ce942174c38e7ea38864cc364a8fe95c66869c85888d812da")

	proof, err := tree.Proof()
	r.NoError(err)
	r.EqualValues(expectedProof, proof)

	/***********************************
	|                4a                |
	|        13              6c        |
	|    9d     .fe.     3d     .6b.   |
	| .00.=01= 02  03 =04=.05. 06  07  |
	***********************************/
}

func TestNewProvingTreeMultiProof2(t *testing.T) {
	r := require.New(t)
	tree := NewProvingTree(GetSha256Parent, []uint64{0, 1, 4})
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("4a2ca61d1fd537170785a8575d424634713c82e7392e67795a807653e498cfd0")
	root, err := tree.Root()
	r.NoError(err)
	r.Equal(expectedRoot, root)

	expectedProof := make([][]byte, 3)
	expectedProof[0], _ = NewNodeFromHex("fe6d3d3bb5dd778af1128cc7b2b33668d51b9a52dfc8f2342be37ddc06a0072d")
	expectedProof[1], _ = NewNodeFromHex("0500000000000000")
	expectedProof[2], _ = NewNodeFromHex("6b2e10cb2111114ce942174c38e7ea38864cc364a8fe95c66869c85888d812da")

	proof, err := tree.Proof()
	r.NoError(err)
	r.EqualValues(expectedProof, proof)

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

// Caching tests:

type sliceWriter struct {
	slice [][]byte
}

func (s *sliceWriter) Write(p []byte) (n int, err error) {
	s.slice = append(s.slice, p)
	return len(p), nil
}

func TestNewCachingTree(t *testing.T) {
	r := require.New(t)
	sliceWriters := make(map[uint]*sliceWriter)
	for i := uint(0); i < 4; i++ {
		sliceWriters[i] = &sliceWriter{}
	}
	cache := make(map[uint]io.Writer)
	for k, v := range sliceWriters {
		cache[k] = v
	}
	tree := NewCachingTree(GetSha256Parent, cache)
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("4a2ca61d1fd537170785a8575d424634713c82e7392e67795a807653e498cfd0")
	root, err := tree.Root()
	r.NoError(err)
	r.Equal(expectedRoot, root)

	r.Len(sliceWriters[0].slice, 8)
	r.Len(sliceWriters[1].slice, 4)
	r.Len(sliceWriters[2].slice, 2)
	r.Len(sliceWriters[3].slice, 1)
	r.Equal(sliceWriters[3].slice[0], expectedRoot)
}

func BenchmarkNewCachingTreeSmall(b *testing.B) {
	var size uint64 = 1 << 23
	cache := make(map[uint]io.Writer)
	for i := uint(7); i < 23; i++ {
		cache[i] = &sliceWriter{}
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
