package cache

func MakeSliceReadWriterFactory() LayerFactory {
	return func(layerHeight uint) LayerReadWriter {
		return &SliceReadWriter{}
	}
}

func MakeSpecificLayersFactory(readWriters map[uint]LayerReadWriter) LayerFactory {
	return func(layerHeight uint) LayerReadWriter {
		return readWriters[layerHeight]
	}
}
