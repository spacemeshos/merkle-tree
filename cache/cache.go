package cache

import (
	"errors"
	"fmt"
	"io"
	"math"
)

const NodeSize = 32

type Writer struct {
	*cache
}

func NewWriterWithLayerFactories(layerFactories []LayerFactory) *Writer {
	return &Writer{
		cache: &cache{
			layers:         make(map[uint]LayerReadWriter),
			layerFactories: layerFactories,
		},
	}
}

func (c *Writer) SetLayer(layer uint, rw LayerReadWriter) {
	c.layers[layer] = rw
}

func (c *Writer) GetLayerWriter(layer uint) LayerWriter {
	layerReadWriter, found := c.layers[layer]
	if !found {
		layerReadWriter = c.fillLayerIfNeeded(layer)
	}
	return layerReadWriter
}

func (c *Writer) SetHash(hashFunc func(lChild, rChild []byte) []byte) {
	c.hash = hashFunc
}

func (c *Writer) GetReader() (*Reader, error) {
	err := c.validateStructure()
	if err != nil {
		return nil, err
	}
	return &Reader{c.cache}, nil
}

type Reader struct {
	*cache
}

type cache struct {
	layers         map[uint]LayerReadWriter
	hash           func(lChild, rChild []byte) []byte
	layerFactories []LayerFactory
}

func (c *Reader) GetLayerReader(layer uint) LayerReader {
	return c.layers[layer]
}

func (c *Reader) GetHashFunc() func(lChild, rChild []byte) []byte {
	return c.hash
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

func (c *cache) fillLayerIfNeeded(layer uint) LayerReadWriter {
	for _, factory := range c.layerFactories {
		layerReadWriter := factory(layer)
		if layerReadWriter != nil {
			c.layers[layer] = layerReadWriter
			return layerReadWriter
		}
	}
	return nil
}

type LayerFactory func(layer uint) LayerReadWriter

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
	io.Writer
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
