package cache

import (
	"errors"
	"github.com/stretchr/testify/require"
	"testing"
)

var someError = errors.New("some error")

type widthReader struct{ width uint64 }

func (r widthReader) Seek(index uint64) error           { return nil }
func (r widthReader) ReadNext() ([]byte, error)         { return nil, someError }
func (r widthReader) Width() uint64                     { return r.width }
func (r widthReader) Write(p []byte) (n int, err error) { panic("implement me") }

func TestCache_ValidateStructure(t *testing.T) {
	r := require.New(t)
	var readers map[uint]LayerReadWriter

	treeCache := &cache{layers: readers}
	err := treeCache.validateStructure()

	r.Error(err)
	r.Equal("reader for base layer must be included", err.Error())
}

func TestCache_ValidateStructure2(t *testing.T) {
	r := require.New(t)
	readers := make(map[uint]LayerReadWriter)

	treeCache := &cache{layers: readers}
	err := treeCache.validateStructure()

	r.Error(err)
	r.Equal("reader for base layer must be included", err.Error())
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

	r.Error(err)
	r.Equal("reader at layer 1 has width 2 instead of 1", err.Error())
}

func TestCache_ValidateStructureFail2(t *testing.T) {
	r := require.New(t)
	readers := make(map[uint]LayerReadWriter)

	readers[0] = widthReader{width: 4}
	readers[1] = widthReader{width: 1}
	readers[2] = widthReader{width: 1}
	treeCache := &cache{layers: readers}
	err := treeCache.validateStructure()

	r.Error(err)
	r.Equal("reader at layer 1 has width 1 instead of 2", err.Error())
}
