package cache

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
)

const NodeSize = 32

type LayerFactory func(layer uint) LayerReadWriter

type Cache struct {
	layers         map[uint]LayerReadWriter
	Hash           func(lChild, rChild []byte) []byte
	layerFactories []LayerFactory
}

func NewCacheWithLayerFactories(layerFactories []LayerFactory) *Cache {
	return &Cache{
		layers:         make(map[uint]LayerReadWriter),
		layerFactories: layerFactories,
	}
}

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

func (c *Cache) GetLayerReader(layer uint) LayerReader {
	return c.layers[layer]
}

func (c *Cache) GetLayerWriter(layer uint) LayerWriter {
	layerReadWriter, found := c.layers[layer]
	if !found {
		layerReadWriter = c.fillLayerIfNeeded(layer)
	}
	return layerReadWriter
}

func (c *Cache) LeafReader() LayerReadWriter {
	return c.layers[0]
}

func (c *Cache) Print(bottom, top int) {
	for i := top; i >= bottom; i-- {
		print("| ")
		sliceReadWriter, ok := c.layers[uint(i)].(*SliceReadWriter)
		if !ok {
			println("-- layer is not a SliceReadWriter --")
			continue
		}
		for _, n := range sliceReadWriter.slice {
			printSpaces(numSpaces(i))
			fmt.Print(hex.EncodeToString(n[:2]))
			printSpaces(numSpaces(i))
		}
		println(" |")
	}
}

func RootHeightFromWidth(width uint64) uint {
	return uint(math.Ceil(math.Log2(float64(width))))
}

func (c *Cache) ValidateStructure() error {
	// Verify we got the base layer.
	if _, found := c.layers[0]; !found {
		return errors.New("reader for base layer must be included")
	}
	width := c.layers[0].Width()
	height := RootHeightFromWidth(width)
	for i := uint(0); i < height; i++ {
		if _, found := c.layers[i]; found && c.layers[i].Width() != width {
			return fmt.Errorf("reader at layer %d has width %d instead of %d", i, c.layers[i].Width(), width)
		}
		width >>= 1
	}
	return nil
}

func (c *Cache) fillLayerIfNeeded(layer uint) LayerReadWriter {
	for _, factory := range c.layerFactories {
		layerReadWriter := factory(layer)
		if layerReadWriter != nil {
			c.layers[layer] = layerReadWriter
			return layerReadWriter
		}
	}
	return nil
}

func numSpaces(n int) int {
	res := 1
	for i := 0; i < n; i++ {
		res += 3 * (1 << uint(i))
	}
	return res
}

func printSpaces(n int) {
	for i := 0; i < n; i++ {
		print(" ")
	}
}
