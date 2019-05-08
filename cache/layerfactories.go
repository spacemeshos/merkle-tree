package cache

import "github.com/spacemeshos/merkle-tree/cache/readwriters"

func MakeSliceReadWriterFactory() LayerFactory {
	return func(layerHeight uint) (LayerReadWriter, error) {
		return &readwriters.SliceReadWriter{}, nil
	}
}

func MakeSpecificLayersFactory(readWriters map[uint]LayerReadWriter) LayerFactory {
	return func(layerHeight uint) (LayerReadWriter, error) {
		return readWriters[layerHeight], nil
	}
}
