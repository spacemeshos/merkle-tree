package cache

import (
	"github.com/spacemeshos/merkle-tree/cache/readwriters"
	"github.com/stretchr/testify/require"
	"io"
	"testing"
)

func TestGroupLayers(t *testing.T) {
	r := require.New(t)

	// Create 9 nodes.
	nodes := genNodes(9)

	// Split the nodes into 3 separate layers.
	layers := make([]LayerReadWriter, 3)
	layers[0] = &readwriters.SliceReadWriter{}
	_, _ = layers[0].Append(nodes[0])
	_, _ = layers[0].Append(nodes[1])
	_, _ = layers[0].Append(nodes[2])
	layers[1] = &readwriters.SliceReadWriter{}
	_, _ = layers[1].Append(nodes[3])
	_, _ = layers[1].Append(nodes[4])
	_, _ = layers[1].Append(nodes[5])
	layers[2] = &readwriters.SliceReadWriter{}
	_, _ = layers[2].Append(nodes[6])
	_, _ = layers[2].Append(nodes[7])
	_, _ = layers[2].Append(nodes[8])

	// Group the layers.
	layer, err := groupLayers(layers)
	r.NoError(err)

	width, err := layer.Width()
	r.NoError(err)
	r.Equal(width, uint64(len(nodes)))

	// Iterate over the layer.
	for _, node := range nodes {
		val, err := layer.ReadNext()
		r.NoError(err)
		r.Equal(val, node)
	}

	// Iterate over the layer with Seek.
	for i, node := range nodes {
		err := layer.Seek(uint64(i))
		r.NoError(err)
		val, err := layer.ReadNext()
		r.NoError(err)
		r.Equal(val, node)
	}
	_, err = layer.ReadNext()
	r.Equal(err, io.EOF)

	// Iterate over the layer with Seek in reverse.
	for i := len(nodes) - 1; i >= 0; i-- {
		err := layer.Seek(uint64(i))
		r.NoError(err)
		val, err := layer.ReadNext()
		r.NoError(err)
		r.Equal(val, nodes[i])
	}

	// Verify that deactivated chunk position is being reset.
	// (target chunk 1 position 1)
	err = layer.Seek(uint64(3))
	r.NoError(err)
	val, err := layer.ReadNext()
	r.NoError(err)
	r.Equal(val, nodes[3])
	// (target chunk 0 position 2)
	err = layer.Seek(uint64(2))
	r.NoError(err)
	val, err = layer.ReadNext()
	r.NoError(err)
	r.Equal(val, nodes[2])
	// (target chunk 1 position 0)
	val, err = layer.ReadNext()
	r.NoError(err)
	r.Equal(val, nodes[3])
}

func TestGroupLayersWithShorterLastLayer(t *testing.T) {
	r := require.New(t)

	// Create 7 nodes.
	nodes := genNodes(7)

	// Split the nodes into 3 separate layers in groups of [3,3,1].
	layers := make([]LayerReadWriter, 3)
	layers[0] = &readwriters.SliceReadWriter{}
	_, _ = layers[0].Append(nodes[0])
	_, _ = layers[0].Append(nodes[1])
	_, _ = layers[0].Append(nodes[2])
	layers[1] = &readwriters.SliceReadWriter{}
	_, _ = layers[1].Append(nodes[3])
	_, _ = layers[1].Append(nodes[4])
	_, _ = layers[1].Append(nodes[5])
	layers[2] = &readwriters.SliceReadWriter{}
	_, _ = layers[2].Append(nodes[6])

	// Group the layers.
	layer, err := groupLayers(layers)
	r.NoError(err)

	width, err := layer.Width()
	r.NoError(err)
	r.Equal(width, uint64(len(nodes)))

	// Iterate over the layer.
	for _, node := range nodes {
		val, err := layer.ReadNext()
		r.NoError(err)
		r.Equal(val, node)
	}

	// Arrive to EOF with ReadNext.
	err = layer.Seek(uint64(6))
	r.NoError(err)
	val, err := layer.ReadNext()
	r.NoError(err)
	r.Equal(val, nodes[6])
	val, err = layer.ReadNext()
	r.Equal(io.EOF, err)

	// Arrive to EOF with Seek.
	err = layer.Seek(uint64(7))
	r.Equal(io.EOF, err)
	err = layer.Seek(uint64(666))
	r.Equal(io.EOF, err)
}

func TestGroupLayersWithShorterMidLayer(t *testing.T) {
	r := require.New(t)

	// Create 7 nodes.
	nodes := genNodes(7)

	// Split the nodes into 3 separate layers in groups of [3,1,3].
	layers := make([]LayerReadWriter, 3)
	layers[0] = &readwriters.SliceReadWriter{}
	_, _ = layers[0].Append(nodes[0])
	_, _ = layers[0].Append(nodes[1])
	_, _ = layers[0].Append(nodes[2])
	layers[1] = &readwriters.SliceReadWriter{}
	_, _ = layers[1].Append(nodes[3])
	layers[2] = &readwriters.SliceReadWriter{}
	_, _ = layers[2].Append(nodes[4])
	_, _ = layers[2].Append(nodes[5])
	_, _ = layers[2].Append(nodes[6])

	// Group the layers.
	_, err := groupLayers(layers)
	r.Equal("layers width mismatch", err.Error())
}
