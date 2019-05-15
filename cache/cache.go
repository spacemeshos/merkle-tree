package cache

import (
	"errors"
	"fmt"
	"github.com/spacemeshos/merkle-tree/shared"
)

const NodeSize = shared.NodeSize

type (
	HashFunc        = shared.HashFunc
	LayerWriter     = shared.LayerWriter
	LayerReader     = shared.LayerReader
	LayerReadWriter = shared.LayerReadWriter
	CacheWriter     = shared.CacheWriter
	CacheReader     = shared.CacheReader
	LayerFactory    = shared.LayerFactory
	CachingPolicy   = shared.CachingPolicy
)

var RootHeightFromWidth = shared.RootHeightFromWidth

type Writer struct {
	*cache
}

// A compile time check to ensure that Writer fully implements CacheWriter.
var _ CacheWriter = (*Writer)(nil)

func NewWriter(shouldCacheLayer CachingPolicy, generateLayer LayerFactory) *Writer {
	return &Writer{
		cache: &cache{
			layers:           make(map[uint]LayerReadWriter),
			generateLayer:    generateLayer,
			shouldCacheLayer: shouldCacheLayer,
		},
	}
}

func (c *Writer) SetLayer(layerHeight uint, rw LayerReadWriter) {
	c.layers[layerHeight] = rw
}

func (c *Writer) GetLayerWriter(layerHeight uint) (LayerWriter, error) {
	layerReadWriter, found := c.layers[layerHeight]
	if !found && c.shouldCacheLayer(layerHeight) {
		var err error
		layerReadWriter, err = c.generateLayer(layerHeight)
		if err != nil {
			return nil, err
		}
		c.layers[layerHeight] = layerReadWriter
	}
	return layerReadWriter, nil
}

func (c *Writer) SetHash(hashFunc HashFunc) {
	c.hash = hashFunc
}

// GetReader returns a cache reader that can be passed into GenerateProof. It first flushes the layer writers to support
// layer writers that have internal buffers that may not be reflected in the reader until flushed. After flushing, this
// method validates the structure of the cache, including that a base layer is cached.
func (c *Writer) GetReader() (CacheReader, error) {
	if err := c.flush(); err != nil {
		return nil, err
	}
	if err := c.validateStructure(); err != nil {
		return nil, err
	}
	return &Reader{c.cache}, nil
}

func (c *Writer) flush() error {
	var lastErr error
	for _, layer := range c.layers {
		lastErr = layer.Flush()
	}
	return lastErr
}

type Reader struct {
	*cache
}

// A compile time check to ensure that Reader fully implements CacheReader.
var _ CacheReader = (*Reader)(nil)

func (c *Reader) Layers() map[uint]LayerReadWriter {
	return c.layers
}

func (c *Reader) GetLayerReader(layerHeight uint) LayerReader {
	return c.layers[layerHeight]
}

func (c *Reader) GetHashFunc() HashFunc {
	return c.hash
}

func (c *Reader) GetLayerFactory() LayerFactory {
	return c.generateLayer
}

func (c *Reader) GetCachingPolicy() CachingPolicy {
	return c.shouldCacheLayer
}

type cache struct {
	layers           map[uint]LayerReadWriter
	hash             HashFunc
	shouldCacheLayer CachingPolicy
	generateLayer    LayerFactory
}

func (c *cache) validateStructure() error {
	// Verify we got the base layer.
	if _, found := c.layers[0]; !found {
		return errors.New("reader for base layer must be included")
	}
	width, err := c.layers[0].Width()
	if err != nil {
		return fmt.Errorf("while getting base layer width: %v", err)
	}
	if width == 0 {
		return errors.New("base layer cannot be empty")
	}
	height := RootHeightFromWidth(width)
	for i := uint(0); i < height; i++ {
		layer, found := c.layers[i]
		if found {
			iWidth, err := layer.Width()
			if err != nil {
				return fmt.Errorf("failed to get width for layer %d: %v", i, err)
			}
			if iWidth != width {
				return fmt.Errorf("reader at layer %d has width %d instead of %d", i, iWidth, width)
			}
		}
		width >>= 1
	}
	return nil
}

//func (c *cache) Print(bottom, top int) {
//	for i := top; i >= bottom; i-- {
//		print("| ")
//		sliceReadWriter, ok := c.layers[uint(i)].(*SliceReadWriter)
//		if !ok {
//			println("-- layer is not a SliceReadWriter --")
//			continue
//		}
//		for _, n := range sliceReadWriter.slice {
//			printSpaces(numSpaces(i))
//			fmt.Print(hex.EncodeToString(n[:2]))
//			printSpaces(numSpaces(i))
//		}
//		println(" |")
//	}
//}
//
//func numSpaces(n int) int {
//	res := 1
//	for i := 0; i < n; i++ {
//		res += 3 * (1 << uint(i))
//	}
//	return res
//}
//
//func printSpaces(n int) {
//	for i := 0; i < n; i++ {
//		print(" ")
//	}
//}
