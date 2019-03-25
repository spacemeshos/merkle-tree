package cache

func MinHeightPolicy(minHeight uint) CachingPolicy {
	return func(layerHeight uint) (shouldCacheLayer bool) {
		return layerHeight >= minHeight
	}
}

func SpecificLayersPolicy(layersToCache map[uint]bool) CachingPolicy {
	return func(layerHeight uint) (shouldCacheLayer bool) {
		return layersToCache[layerHeight]
	}
}

func Combine(first, second CachingPolicy) CachingPolicy {
	return func(layerHeight uint) (shouldCacheLayer bool) {
		return first(layerHeight) || second(layerHeight)
	}
}
