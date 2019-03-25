package cache

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMakeMemoryReadWriterFactory(t *testing.T) {
	r := require.New(t)
	treeCache := NewWriterWithLayerFactories([]LayerFactory{
		MakeMemoryReadWriterFactory(2),
	})
	treeCache.SetLayer(0, widthReader{width: 1})

	cacheReader, err := treeCache.GetReader()
	r.NoError(err)

	reader := cacheReader.GetLayerReader(1)
	r.Nil(reader)
	reader = cacheReader.GetLayerReader(2)
	r.Nil(reader)
	reader = cacheReader.GetLayerReader(3)
	r.Nil(reader)

	writer := treeCache.GetLayerWriter(1)
	r.Nil(writer)
	writer = treeCache.GetLayerWriter(2)
	r.NotNil(writer)
	writer = treeCache.GetLayerWriter(3)
	r.NotNil(writer)

	cacheReader, err = treeCache.GetReader()
	r.NoError(err)

	reader = cacheReader.GetLayerReader(1)
	r.Nil(reader)
	reader = cacheReader.GetLayerReader(2)
	r.NotNil(reader)
	reader = cacheReader.GetLayerReader(3)
	r.NotNil(reader)
}

func TestMakeMemoryReadWriterFactoryForLayers(t *testing.T) {
	r := require.New(t)
	treeCache := NewWriterWithLayerFactories([]LayerFactory{
		MakeMemoryReadWriterFactoryForLayers([]uint{1, 3}),
	})
	treeCache.SetLayer(0, widthReader{width: 1})

	cacheReader, err := treeCache.GetReader()
	r.NoError(err)

	reader := cacheReader.GetLayerReader(1)
	r.Nil(reader)
	reader = cacheReader.GetLayerReader(2)
	r.Nil(reader)
	reader = cacheReader.GetLayerReader(3)
	r.Nil(reader)

	writer := treeCache.GetLayerWriter(1)
	r.NotNil(writer)
	writer = treeCache.GetLayerWriter(2)
	r.Nil(writer)
	writer = treeCache.GetLayerWriter(3)
	r.NotNil(writer)

	cacheReader, err = treeCache.GetReader()
	r.NoError(err)

	reader = cacheReader.GetLayerReader(1)
	r.NotNil(reader)
	reader = cacheReader.GetLayerReader(2)
	r.Nil(reader)
	reader = cacheReader.GetLayerReader(3)
	r.NotNil(reader)
}

func TestMakeSpecificLayerFactory(t *testing.T) {
	r := require.New(t)
	readWriter := &SliceReadWriter{}
	treeCache := NewWriterWithLayerFactories([]LayerFactory{
		MakeSpecificLayerFactory(1, readWriter),
	})
	treeCache.SetLayer(0, widthReader{width: 1})

	cacheReader, err := treeCache.GetReader()
	r.NoError(err)

	reader := cacheReader.GetLayerReader(1)
	r.Nil(reader)
	reader = cacheReader.GetLayerReader(2)
	r.Nil(reader)

	writer := treeCache.GetLayerWriter(1)
	r.Equal(readWriter, writer)
	writer = treeCache.GetLayerWriter(2)
	r.Nil(writer)

	cacheReader, err = treeCache.GetReader()
	r.NoError(err)

	reader = cacheReader.GetLayerReader(1)
	r.Equal(readWriter, reader)
	reader = cacheReader.GetLayerReader(2)
	r.Nil(reader)
}
