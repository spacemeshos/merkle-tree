package merkle

import (
	"encoding/hex"
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
	sliceReadWriters := make(map[uint]*sliceReadWriter)
	sliceReadWriters[0] = &sliceReadWriter{}
	sliceReadWriters[1] = &sliceReadWriter{}
	sliceReadWriters[2] = &sliceReadWriter{}
	leavesToProve := []uint64{0, 4, 7}

	tree := NewTreeBuilder(GetSha256Parent).
		WithCache(WritersFromSliceReadWriters(sliceReadWriters)).
		WithLeavesToProve(leavesToProve).
		Build()
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("89a0f1577268cc19b0a39c7a69f804fd140640c699585eb635ebb03c06154cce")
	root := tree.Root()
	r.Equal(expectedRoot, root)

	r.Len(sliceReadWriters[0].slice, 8)
	r.Len(sliceReadWriters[1].slice, 4)
	r.Len(sliceReadWriters[2].slice, 2)

	var proof, expectedProof nodes
	var err error
	proof, err = GenerateProof(leavesToProve, NodeReadersFromSliceReadWriters(sliceReadWriters), GetSha256Parent)
	r.NoError(err)

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof, "actual")
}

func BenchmarkGenerateProof(b *testing.B) {
	const treeHeight = 23
	r := require.New(b)
	sliceReadWriters := make(map[uint]*sliceReadWriter)
	sliceReadWriters[0] = &sliceReadWriter{}
	for i := 7; i < treeHeight; i++ {
		sliceReadWriters[uint(i)] = &sliceReadWriter{}
	}
	var leavesToProve []uint64
	for i := 0; i < 20; i++ {
		leavesToProve = append(leavesToProve, uint64(i)*400000)
	}

	tree := NewTreeBuilder(GetSha256Parent).
		WithCache(WritersFromSliceReadWriters(sliceReadWriters)).
		WithLeavesToProve(leavesToProve).
		Build()
	for i := uint64(0); i < 1<<treeHeight; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}

	r.Len(sliceReadWriters[0].slice, 1<<treeHeight)

	var proof, expectedProof nodes
	var err error

	start := time.Now()
	proof, err = GenerateProof(leavesToProve, NodeReadersFromSliceReadWriters(sliceReadWriters), GetSha256Parent)
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
	sliceReadWriters := make(map[uint]*sliceReadWriter)
	sliceReadWriters[0] = &sliceReadWriter{}
	sliceReadWriters[1] = &sliceReadWriter{}
	sliceReadWriters[2] = &sliceReadWriter{}
	sliceReadWriters[3] = &sliceReadWriter{}
	leavesToProve := []uint64{0, 4, 7}
	tree := NewTreeBuilder(GetSha256Parent).
		WithCache(WritersFromSliceReadWriters(sliceReadWriters)).
		WithLeavesToProve(leavesToProve).
		Build()
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("89a0f1577268cc19b0a39c7a69f804fd140640c699585eb635ebb03c06154cce")
	root := tree.Root()
	r.Equal(expectedRoot, root)

	r.Len(sliceReadWriters[0].slice, 8)
	r.Len(sliceReadWriters[1].slice, 4)
	r.Len(sliceReadWriters[2].slice, 2)
	r.Len(sliceReadWriters[3].slice, 1)
	r.Equal(expectedRoot, sliceReadWriters[3].slice[0])

	var proof, expectedProof nodes
	var err error
	proof, err = GenerateProof(leavesToProve, NodeReadersFromSliceReadWriters(sliceReadWriters), GetSha256Parent)
	r.NoError(err)

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof, "actual")
}

func TestGenerateProofWithoutCache(t *testing.T) {
	r := require.New(t)
	sliceReadWriters := make(map[uint]*sliceReadWriter)
	sliceReadWriters[0] = &sliceReadWriter{}
	leavesToProve := []uint64{0, 4, 7}
	tree := NewTreeBuilder(GetSha256Parent).
		WithCache(WritersFromSliceReadWriters(sliceReadWriters)).
		WithLeavesToProve(leavesToProve).
		Build()
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("89a0f1577268cc19b0a39c7a69f804fd140640c699585eb635ebb03c06154cce")
	root := tree.Root()
	r.Equal(expectedRoot, root)

	r.Len(sliceReadWriters[0].slice, 8)

	var proof, expectedProof nodes
	var err error
	proof, err = GenerateProof(leavesToProve, NodeReadersFromSliceReadWriters(sliceReadWriters), GetSha256Parent)
	r.NoError(err)

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof, "actual")
}

func TestGenerateProofWithSingleLayerCache(t *testing.T) {
	r := require.New(t)
	sliceReadWriters := make(map[uint]*sliceReadWriter)
	sliceReadWriters[0] = &sliceReadWriter{}
	sliceReadWriters[2] = &sliceReadWriter{}
	leavesToProve := []uint64{0, 4, 7}
	tree := NewTreeBuilder(GetSha256Parent).
		WithCache(WritersFromSliceReadWriters(sliceReadWriters)).
		WithLeavesToProve(leavesToProve).
		Build()
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("89a0f1577268cc19b0a39c7a69f804fd140640c699585eb635ebb03c06154cce")
	root := tree.Root()
	r.Equal(expectedRoot, root)

	r.Len(sliceReadWriters[0].slice, 8)
	r.Len(sliceReadWriters[2].slice, 2)

	var proof, expectedProof nodes
	var err error
	proof, err = GenerateProof(leavesToProve, NodeReadersFromSliceReadWriters(sliceReadWriters), GetSha256Parent)
	r.NoError(err)

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof)
}
func TestGenerateProofWithSingleLayerCache2(t *testing.T) {
	r := require.New(t)
	sliceReadWriters := make(map[uint]*sliceReadWriter)
	sliceReadWriters[0] = &sliceReadWriter{}
	sliceReadWriters[1] = &sliceReadWriter{}
	leavesToProve := []uint64{0, 4, 7}
	tree := NewTreeBuilder(GetSha256Parent).
		WithCache(WritersFromSliceReadWriters(sliceReadWriters)).
		WithLeavesToProve(leavesToProve).
		Build()
	for i := uint64(0); i < 8; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}
	expectedRoot, _ := NewNodeFromHex("89a0f1577268cc19b0a39c7a69f804fd140640c699585eb635ebb03c06154cce")
	root := tree.Root()
	r.Equal(expectedRoot, root)

	r.Len(sliceReadWriters[0].slice, 8)
	r.Len(sliceReadWriters[1].slice, 4)

	var proof, expectedProof nodes
	var err error
	proof, err = GenerateProof(leavesToProve, NodeReadersFromSliceReadWriters(sliceReadWriters), GetSha256Parent)
	r.NoError(err)

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof)
}

func TestGenerateProofUnbalanced(t *testing.T) {
	r := require.New(t)
	sliceReadWriters := make(map[uint]*sliceReadWriter)
	sliceReadWriters[0] = &sliceReadWriter{}
	sliceReadWriters[1] = &sliceReadWriter{}
	sliceReadWriters[2] = &sliceReadWriter{}
	leavesToProve := []uint64{0, 4, 6}

	tree := NewTreeBuilder(GetSha256Parent).
		WithCache(WritersFromSliceReadWriters(sliceReadWriters)).
		WithLeavesToProve(leavesToProve).
		Build()
	for i := uint64(0); i < 7; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}

	r.Len(sliceReadWriters[0].slice, 7)
	r.Len(sliceReadWriters[1].slice, 3)
	r.Len(sliceReadWriters[2].slice, 1)

	var proof, expectedProof nodes
	var err error
	proof, err = GenerateProof(leavesToProve, NodeReadersFromSliceReadWriters(sliceReadWriters), GetSha256Parent)
	r.NoError(err)

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof, "actual")
}

func TestGenerateProofUnbalanced2(t *testing.T) {
	r := require.New(t)
	sliceReadWriters := make(map[uint]*sliceReadWriter)
	sliceReadWriters[0] = &sliceReadWriter{}
	sliceReadWriters[1] = &sliceReadWriter{}
	sliceReadWriters[2] = &sliceReadWriter{}
	leavesToProve := []uint64{0, 4}

	tree := NewTreeBuilder(GetSha256Parent).
		WithCache(WritersFromSliceReadWriters(sliceReadWriters)).
		WithLeavesToProve(leavesToProve).
		Build()
	for i := uint64(0); i < 6; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}

	r.Len(sliceReadWriters[0].slice, 6)
	r.Len(sliceReadWriters[1].slice, 3)
	r.Len(sliceReadWriters[2].slice, 1)

	var proof, expectedProof nodes
	var err error
	proof, err = GenerateProof(leavesToProve, NodeReadersFromSliceReadWriters(sliceReadWriters), GetSha256Parent)
	r.NoError(err)

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof, "actual")
}

func TestGenerateProofUnbalanced3(t *testing.T) {
	r := require.New(t)
	sliceReadWriters := make(map[uint]*sliceReadWriter)
	sliceReadWriters[0] = &sliceReadWriter{}
	sliceReadWriters[1] = &sliceReadWriter{}
	sliceReadWriters[2] = &sliceReadWriter{}
	leavesToProve := []uint64{0}

	tree := NewTreeBuilder(GetSha256Parent).
		WithCache(WritersFromSliceReadWriters(sliceReadWriters)).
		WithLeavesToProve(leavesToProve).
		Build()
	for i := uint64(0); i < 7; i++ {
		err := tree.AddLeaf(NewNodeFromUint64(i))
		r.NoError(err)
	}

	r.Len(sliceReadWriters[0].slice, 7)
	r.Len(sliceReadWriters[1].slice, 3)
	r.Len(sliceReadWriters[2].slice, 1)

	var proof, expectedProof nodes
	var err error
	proof, err = GenerateProof(leavesToProve, NodeReadersFromSliceReadWriters(sliceReadWriters), GetSha256Parent)
	r.NoError(err)

	expectedProof = tree.Proof()
	r.EqualValues(expectedProof, proof, "actual")
}

type nodes [][]byte

func (n nodes) String() string {
	s := ""
	for _, v := range n {
		s += hex.EncodeToString(v[:2]) + " "
	}
	return s
}

func WritersFromSliceReadWriters(sliceReadWriters map[uint]*sliceReadWriter) map[uint]io.Writer {
	cache := make(map[uint]io.Writer)
	for k, v := range sliceReadWriters {
		cache[k] = v
	}
	return cache
}

func NodeReadersFromSliceReadWriters(sliceReadWriters map[uint]*sliceReadWriter) map[uint]NodeReader {
	nodeReaders := make(map[uint]NodeReader, len(sliceReadWriters))
	for k, v := range sliceReadWriters {
		nodeReaders[k] = v
	}
	return nodeReaders
}
