package merkle

import (
	"errors"
	"github.com/stretchr/testify/require"
	"io"
	"testing"
)

func TestNewTreeCache(t *testing.T) {

}

var someError = errors.New("some error")

type seekErrorReader struct{}

func (seekErrorReader) Seek(index uint64) error   { return someError }
func (seekErrorReader) ReadNext() ([]byte, error) { panic("implement me") }
func (seekErrorReader) Width() uint64             { return 3 }

type readErrorReader struct{}

func (readErrorReader) Seek(index uint64) error   { return nil }
func (readErrorReader) ReadNext() ([]byte, error) { return nil, someError }
func (readErrorReader) Width() uint64             { return 8 }

type seekEOFReader struct{}

func (seekEOFReader) Seek(index uint64) error   { return io.EOF }
func (seekEOFReader) ReadNext() ([]byte, error) { panic("implement me") }
func (seekEOFReader) Width() uint64             { return 1 }

type widthReader struct{ width uint64 }

func (r widthReader) Seek(index uint64) error   { return nil }
func (r widthReader) ReadNext() ([]byte, error) { return nil, someError }
func (r widthReader) Width() uint64             { return r.width }

func TestNewTreeCacheErrors(t *testing.T) {
	r := require.New(t)
	var readers map[uint]NodeReader

	treeCache, err := NewTreeCache(readers, GetSha256Parent)

	r.Error(err)
	r.Equal("reader for base layer must be included", err.Error())
	r.Nil(treeCache)
}

func TestNewTreeCache2(t *testing.T) {
	r := require.New(t)
	readers := make(map[uint]NodeReader)

	treeCache, err := NewTreeCache(readers, GetSha256Parent)

	r.Error(err)
	r.Equal("reader for base layer must be included", err.Error())
	r.Nil(treeCache)
}

func TestNewTreeCache3(t *testing.T) {
	r := require.New(t)
	readers := make(map[uint]NodeReader)

	readers[0] = seekErrorReader{}
	treeCache, err := NewTreeCache(readers, GetSha256Parent)

	r.NoError(err)
	r.NotNil(treeCache)

	nodePos := position{}
	node, err := treeCache.GetNode(nodePos)

	r.Error(err)
	r.Equal("while seeking to position <h: 0 i: 0> in cache: some error", err.Error())
	r.Nil(node)

}

func TestNewTreeCache4(t *testing.T) {
	r := require.New(t)
	readers := make(map[uint]NodeReader)

	readers[0] = readErrorReader{}
	treeCache, err := NewTreeCache(readers, GetSha256Parent)

	r.NoError(err)
	r.NotNil(treeCache)

	nodePos := position{}
	node, err := treeCache.GetNode(nodePos)

	r.Error(err)
	r.Equal("while reading from cache: some error", err.Error())
	r.Nil(node)
}

func TestNewTreeCache5(t *testing.T) {
	r := require.New(t)
	readers := make(map[uint]NodeReader)

	readers[0] = seekErrorReader{}
	readers[1] = seekEOFReader{}
	treeCache, err := NewTreeCache(readers, GetSha256Parent)

	r.NoError(err)
	r.NotNil(treeCache)

	nodePos := position{height: 1}
	node, err := treeCache.GetNode(nodePos)

	r.Error(err)
	r.Equal("while seeking to position <h: 0 i: 0> in cache: some error", err.Error())
	r.Nil(node)
}

func TestNewTreeCache6(t *testing.T) {
	r := require.New(t)
	readers := make(map[uint]NodeReader)

	readers[0] = seekErrorReader{}
	readers[1] = widthReader{width: 1}
	readers[2] = seekEOFReader{}
	treeCache, err := NewTreeCache(readers, GetSha256Parent)

	r.NoError(err)
	r.NotNil(treeCache)

	nodePos := position{height: 2}
	node, err := treeCache.GetNode(nodePos)

	r.Error(err)
	r.Equal("while calculating ephemeral node at position <h: 1 i: 1>: while seeking to position <h: 0 i: 10> in cache: some error", err.Error())
	r.Nil(node)
}

func TestNewTreeCache7(t *testing.T) {
	r := require.New(t)
	readers := make(map[uint]NodeReader)

	readers[0] = widthReader{width: 2}
	readers[1] = seekEOFReader{}
	treeCache, err := NewTreeCache(readers, GetSha256Parent)

	r.NoError(err)
	r.NotNil(treeCache)

	nodePos := position{height: 1}
	node, err := treeCache.GetNode(nodePos)

	r.Error(err)
	r.Equal("while traversing subtree for root: while reading a leaf: some error", err.Error())
	r.Nil(node)
}

func TestNewTreeCacheStructureSuccess(t *testing.T) {
	r := require.New(t)
	readers := make(map[uint]NodeReader)

	readers[0] = widthReader{width: 4}
	readers[1] = widthReader{width: 2}
	readers[2] = widthReader{width: 1}
	treeCache, err := NewTreeCache(readers, GetSha256Parent)

	r.NoError(err)
	r.NotNil(treeCache)
}

func TestNewTreeCacheStructureFail(t *testing.T) {
	r := require.New(t)
	readers := make(map[uint]NodeReader)

	readers[0] = widthReader{width: 3}
	readers[1] = widthReader{width: 2}
	readers[2] = widthReader{width: 1}
	treeCache, err := NewTreeCache(readers, GetSha256Parent)

	r.Error(err)
	r.Equal("reader at layer 1 has width 2 instead of 1", err.Error())
	r.Nil(treeCache)
}

func TestNewTreeCacheStructureFail2(t *testing.T) {
	r := require.New(t)
	readers := make(map[uint]NodeReader)

	readers[0] = widthReader{width: 4}
	readers[1] = widthReader{width: 1}
	readers[2] = widthReader{width: 1}
	treeCache, err := NewTreeCache(readers, GetSha256Parent)

	r.Error(err)
	r.Equal("reader at layer 1 has width 1 instead of 2", err.Error())
	r.Nil(treeCache)
}

func TestPosition_isAncestorOf(t *testing.T) {
	lower := position{
		index:  0,
		height: 0,
	}

	higher := position{
		index:  0,
		height: 1,
	}

	isAncestor := lower.isAncestorOf(higher)

	require.False(t, isAncestor)
}
