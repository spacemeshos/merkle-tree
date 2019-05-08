package cache

import (
	"errors"
	"fmt"
	"github.com/spacemeshos/merkle-tree/cache/readwriters"
	"math"
)

const NodeSize = 32

type Writer struct {
	*cache
}

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

func (c *Writer) SetHash(hashFunc func(lChild, rChild []byte) []byte) {
	c.hash = hashFunc
}

// GetReader returns a cache reader that can be passed into GenerateProof. It first flushes the layer writers to support
// layer writers that have internal buffers that may not be reflected in the reader until flushed. After flushing, this
// method validates the structure of the cache, including that a base layer is cached.
func (c *Writer) GetReader() (*Reader, error) {
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

func (c *Reader) GetLayerReader(layerHeight uint) LayerReader {
	return c.layers[layerHeight]
}

func (c *Reader) GetHashFunc() func(lChild, rChild []byte) []byte {
	return c.hash
}

type cache struct {
	layers           map[uint]LayerReadWriter
	hash             func(lChild, rChild []byte) []byte
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

type CachingPolicy func(layerHeight uint) (shouldCacheLayer bool)

type LayerFactory func(layerHeight uint) (LayerReadWriter, error)

// LayerReadWriter is a combined reader-writer. Note that the Seek() method only belongs to the LayerReader interface
// and does not affect the LayerWriter.
type LayerReadWriter interface {
	LayerReader
	LayerWriter
}

var _ LayerReadWriter = &readwriters.FileReadWriter{}
var _ LayerReadWriter = &readwriters.SliceReadWriter{}

type LayerReader interface {
	Seek(index uint64) error
	ReadNext() ([]byte, error)
	Width() (uint64, error)
}

type LayerWriter interface {
	Append(p []byte) (n int, err error)
	Flush() error
}

func RootHeightFromWidth(width uint64) uint {
	return uint(math.Ceil(math.Log2(float64(width))))
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
