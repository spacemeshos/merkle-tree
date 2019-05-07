package cache

import (
	"errors"
	"fmt"
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
	width := c.layers[0].Width()
	if width == 0 {
		return errors.New("base layer cannot be empty")
	}
	height := RootHeightFromWidth(width)
	for i := uint(0); i < height; i++ {
		if _, found := c.layers[i]; found && c.layers[i].Width() != width {
			return fmt.Errorf("reader at layer %d has width %d instead of %d", i, c.layers[i].Width(), width)
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

type LayerReader interface {
	Seek(index uint64) error
	ReadNext() ([]byte, error)
	Width() uint64
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
