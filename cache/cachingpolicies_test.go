package cache

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMakeMemoryReadWriterFactory(t *testing.T) {
	r := require.New(t)
	cacheWriter := NewWriter(MinHeightPolicy(2), MakeSliceReadWriterFactory())
	cacheWriter.SetLayer(0, widthReader{width: 1})

	cacheReader, err := cacheWriter.GetReader()
	r.NoError(err)

	reader := cacheReader.GetLayerReader(1)
	r.Nil(reader)
	reader = cacheReader.GetLayerReader(2)
	r.Nil(reader)
	reader = cacheReader.GetLayerReader(3)
	r.Nil(reader)

	writer := cacheWriter.GetLayerWriter(1)
	r.Nil(writer)
	writer = cacheWriter.GetLayerWriter(2)
	r.NotNil(writer)
	writer = cacheWriter.GetLayerWriter(3)
	r.NotNil(writer)

	cacheReader, err = cacheWriter.GetReader()
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
	cacheWriter := NewWriter(SpecificLayersPolicy(map[uint]bool{1: true, 3: true}), MakeSliceReadWriterFactory())
	cacheWriter.SetLayer(0, widthReader{width: 1})

	cacheReader, err := cacheWriter.GetReader()
	r.NoError(err)

	reader := cacheReader.GetLayerReader(1)
	r.Nil(reader)
	reader = cacheReader.GetLayerReader(2)
	r.Nil(reader)
	reader = cacheReader.GetLayerReader(3)
	r.Nil(reader)

	writer := cacheWriter.GetLayerWriter(1)
	r.NotNil(writer)
	writer = cacheWriter.GetLayerWriter(2)
	r.Nil(writer)
	writer = cacheWriter.GetLayerWriter(3)
	r.NotNil(writer)

	cacheReader, err = cacheWriter.GetReader()
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
	cacheWriter := NewWriter(
		SpecificLayersPolicy(map[uint]bool{1: true}),
		MakeSpecificLayersFactory(map[uint]LayerReadWriter{1: readWriter}),
	)
	cacheWriter.SetLayer(0, widthReader{width: 1})

	cacheReader, err := cacheWriter.GetReader()
	r.NoError(err)

	reader := cacheReader.GetLayerReader(1)
	r.Nil(reader)
	reader = cacheReader.GetLayerReader(2)
	r.Nil(reader)

	writer := cacheWriter.GetLayerWriter(1)
	r.Equal(readWriter, writer)
	writer = cacheWriter.GetLayerWriter(2)
	r.Nil(writer)

	cacheReader, err = cacheWriter.GetReader()
	r.NoError(err)

	reader = cacheReader.GetLayerReader(1)
	r.Equal(readWriter, reader)
	reader = cacheReader.GetLayerReader(2)
	r.Nil(reader)
}
