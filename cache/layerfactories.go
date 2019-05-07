package cache

func MakeSliceReadWriterFactory() LayerFactory {
	return func(layerHeight uint) (LayerReadWriter, error) {
		return &SliceReadWriter{}, nil
	}
}

func MakeSpecificLayersFactory(readWriters map[uint]LayerReadWriter) LayerFactory {
	return func(layerHeight uint) (LayerReadWriter, error) {
		return readWriters[layerHeight], nil
	}
}
