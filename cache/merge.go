package cache

import (
	"errors"
	"github.com/spacemeshos/merkle-tree"
	"io"
)

// Merge merges a slice of caches into one unified cache.
// Layers of all caches per each height are appended and grouped, while
// the hash function, caching policy and layer factory are taken
// from the first cache of the slice.
func Merge(caches []CacheReader) (*Reader, error) {
	if len(caches) < 2 {
		return nil, errors.New("number of caches must be at least 2")
	}

	// Aggregate caches' layers by height.
	layerGroups := make(map[uint][]LayerReadWriter)
	for _, cache := range caches {
		for height, layer := range cache.Layers() {
			layerGroups[height] = append(layerGroups[height], layer)
		}
	}

	// Group layer groups.
	layers := make(map[uint]LayerReadWriter)
	for height, layerGroup := range layerGroups {
		if len(layerGroup) != len(caches) {
			return nil, errors.New("number of layers per height mismatch")
		}

		group, err := groupLayers(layerGroup)
		if err != nil {
			return nil, err
		}
		layers[height] = group
	}

	hashFunc := caches[0].GetHashFunc()
	layerFactory := caches[0].GetLayerFactory()
	cachingPolicy := caches[0].GetCachingPolicy()

	cache := &cache{
		layers:           layers,
		hash:             hashFunc,
		shouldCacheLayer: cachingPolicy,
		generateLayer:    layerFactory,
	}
	return &Reader{cache}, nil
}

// BuildTop builds the top layers of a cache, and returns
// its new version in addition to its root.
func BuildTop(cacheReader CacheReader) (*Reader, []byte, error) {
	// Find the cache highest layer.
	var maxHeight uint
	for height := range cacheReader.Layers() {
		if height > maxHeight {
			maxHeight = height
		}
	}

	// Create an adjusted caching policy for the new subtree.
	newCachingPolicy := func(layerHeight uint) bool {
		return cacheReader.GetCachingPolicy()(maxHeight + layerHeight)
	}

	// Create a subtree with the cache highest layer as its leaves.
	subtreeWriter := NewWriter(newCachingPolicy, cacheReader.GetLayerFactory())
	subtree, err := merkle.NewTreeBuilder().
		WithHashFunc(cacheReader.GetHashFunc()).
		WithCacheWriter(subtreeWriter).
		Build()
	if err != nil {
		return nil, nil, err
	}

	layer := cacheReader.GetLayerReader(maxHeight)
	for {
		val, err := layer.ReadNext()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, nil, err
			}
		}

		err = subtree.AddLeaf(val)
		if err != nil {
			return nil, nil, err
		}
	}

	// Clone the existing cache.
	newCache := &cache{
		layers:           cacheReader.Layers(),
		hash:             cacheReader.GetHashFunc(),
		shouldCacheLayer: cacheReader.GetCachingPolicy(),
		generateLayer:    cacheReader.GetLayerFactory(),
	}

	// Add the subtree cache layers on top of the existing ones.
	for height, layer := range subtreeWriter.layers {
		if height == 0 {
			continue
		}
		newCache.layers[height+maxHeight] = layer
	}

	err = newCache.validateStructure()
	if err != nil {
		return nil, nil, err
	}

	return &Reader{cache: newCache}, subtree.Root(), nil
}
