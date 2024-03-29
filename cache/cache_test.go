package cache

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

var someError = errors.New("some error")

type widthReader struct{ width uint64 }

// A compile time check to ensure that widthReader fully implements LayerReadWriter.
var _ LayerReadWriter = (*widthReader)(nil)

func (r widthReader) Seek(index uint64) error            { return nil }
func (r widthReader) ReadNext() ([]byte, error)          { return nil, someError }
func (r widthReader) Width() (uint64, error)             { return r.width, nil }
func (r widthReader) Append(p []byte) (n int, err error) { panic("implement me") }
func (r widthReader) Flush() error                       { return nil }
func (r widthReader) Close() error                       { return nil }

func TestCache_ValidateStructure(t *testing.T) {
	r := require.New(t)
	var readers map[uint]LayerReadWriter

	treeCache := &cache{layers: readers}
	err := treeCache.validateStructure()

	r.Error(err,"reader for base layer must be included")
}

func TestCache_ValidateStructure2(t *testing.T) {
	r := require.New(t)
	readers := make(map[uint]LayerReadWriter)

	treeCache := &cache{layers: readers}
	err := treeCache.validateStructure()

	r.Error(err,"reader for base layer must be included")
}

func TestCache_ValidateStructureSuccess(t *testing.T) {
	r := require.New(t)
	readers := make(map[uint]LayerReadWriter)

	readers[0] = widthReader{width: 4}
	readers[1] = widthReader{width: 2}
	readers[2] = widthReader{width: 1}
	treeCache := &cache{layers: readers}
	err := treeCache.validateStructure()

	r.NoError(err)
}

func TestCache_ValidateStructureFail(t *testing.T) {
	r := require.New(t)
	readers := make(map[uint]LayerReadWriter)

	readers[0] = widthReader{width: 3}
	readers[1] = widthReader{width: 2}
	readers[2] = widthReader{width: 1}
	treeCache := &cache{layers: readers}
	err := treeCache.validateStructure()

	r.Error(err,"reader at layer 1 has width 2 instead of 1")
}

func TestCache_ValidateStructureFail2(t *testing.T) {
	r := require.New(t)
	readers := make(map[uint]LayerReadWriter)

	readers[0] = widthReader{width: 4}
	readers[1] = widthReader{width: 1}
	readers[2] = widthReader{width: 1}
	treeCache := &cache{layers: readers}
	err := treeCache.validateStructure()

	r.Error(err,"reader at layer 1 has width 1 instead of 2")
}
