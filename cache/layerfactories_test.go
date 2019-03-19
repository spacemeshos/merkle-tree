package cache

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMakeMemoryReadWriterFactory(t *testing.T) {
	r := require.New(t)
	treeCache := NewCacheWithLayerFactories([]LayerFactory{MakeMemoryReadWriterFactory(2)})

	// Layer 0: Reader is empty before and after requesting writer.
	reader := treeCache.GetLayerReader(0)
	r.Nil(reader)
	writer := treeCache.GetLayerWriter(0)
	r.Nil(writer)
	reader = treeCache.GetLayerReader(0)
	r.Nil(reader)

	// Layer 1: Reader is empty before and after requesting writer.
	reader = treeCache.GetLayerReader(1)
	r.Nil(reader)
	writer = treeCache.GetLayerWriter(1)
	r.Nil(writer)
	reader = treeCache.GetLayerReader(1)
	r.Nil(reader)

	// Layer 2: Reader is empty before requesting writer, writer is available and reader is available after requesting
	// writer.
	reader = treeCache.GetLayerReader(2)
	r.Nil(reader)
	writer = treeCache.GetLayerWriter(2)
	r.NotNil(writer)
	reader = treeCache.GetLayerReader(2)
	r.NotNil(reader)

	// Layer 3: Reader is empty before requesting writer, writer is available and reader is available after requesting
	// writer.
	reader = treeCache.GetLayerReader(3)
	r.Nil(reader)
	writer = treeCache.GetLayerWriter(3)
	r.NotNil(writer)
	reader = treeCache.GetLayerReader(3)
	r.NotNil(reader)
}

func TestMakeMemoryReadWriterFactoryForLayers(t *testing.T) {
	r := require.New(t)
	treeCache := NewCacheWithLayerFactories([]LayerFactory{MakeMemoryReadWriterFactoryForLayers([]uint{0, 2})})

	// Layer 0: Reader is empty before requesting writer, writer is available and reader is available after requesting
	// writer.
	reader := treeCache.GetLayerReader(0)
	r.Nil(reader)
	writer := treeCache.GetLayerWriter(0)
	r.NotNil(writer)
	reader = treeCache.GetLayerReader(0)
	r.NotNil(reader)

	// Layer 1: Reader is empty before and after requesting writer.
	reader = treeCache.GetLayerReader(1)
	r.Nil(reader)
	writer = treeCache.GetLayerWriter(1)
	r.Nil(writer)
	reader = treeCache.GetLayerReader(1)
	r.Nil(reader)

	// Layer 2: Reader is empty before requesting writer, writer is available and reader is available after requesting
	// writer.
	reader = treeCache.GetLayerReader(2)
	r.Nil(reader)
	writer = treeCache.GetLayerWriter(2)
	r.NotNil(writer)
	reader = treeCache.GetLayerReader(2)
	r.NotNil(reader)

	// Layer 3: Reader is empty before and after requesting writer.
	reader = treeCache.GetLayerReader(3)
	r.Nil(reader)
	writer = treeCache.GetLayerWriter(3)
	r.Nil(writer)
	reader = treeCache.GetLayerReader(3)
	r.Nil(reader)
}

func TestMakeSpecificLayerFactory(t *testing.T) {
	r := require.New(t)
	readWriter := &SliceReadWriter{}
	treeCache := NewCacheWithLayerFactories([]LayerFactory{MakeSpecificLayerFactory(1, readWriter)})

	// Layer 0: Reader is empty before and after requesting writer.
	reader := treeCache.GetLayerReader(0)
	r.Nil(reader)
	writer := treeCache.GetLayerWriter(0)
	r.Nil(writer)
	reader = treeCache.GetLayerReader(0)
	r.Nil(reader)

	// Layer 1: Reader is empty before, but after requesting writer - both reader and writer are as expected.
	reader = treeCache.GetLayerReader(1)
	r.Nil(reader)
	writer = treeCache.GetLayerWriter(1)
	r.Equal(readWriter, writer)
	reader = treeCache.GetLayerReader(1)
	r.Equal(readWriter, reader)

	// Layer 2: Reader is empty before and after requesting writer.
	reader = treeCache.GetLayerReader(2)
	r.Nil(reader)
	writer = treeCache.GetLayerWriter(2)
	r.Nil(writer)
	reader = treeCache.GetLayerReader(2)
	r.Nil(reader)
}
