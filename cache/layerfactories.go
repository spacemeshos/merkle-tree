package cache

func MakeMemoryReadWriterFactory(minLayer uint) LayerFactory {
	return func(layer uint) LayerReadWriter {
		if layer >= minLayer {
			return &SliceReadWriter{}
		}
		return nil
	}
}

func MakeMemoryReadWriterFactoryForLayers(layers []uint) LayerFactory {
	return func(layer uint) LayerReadWriter {
		for _, l := range layers {
			if layer == l {
				return &SliceReadWriter{}
			}
		}
		return nil
	}
}

func MakeSpecificLayerFactory(layer uint, readWriter LayerReadWriter) LayerFactory {
	return func(l uint) LayerReadWriter {
		if layer == l {
			return readWriter
		}
		return nil
	}
}
