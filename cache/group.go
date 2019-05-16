package cache

import (
	"errors"
	"io"
)

type GroupLayerReadWriter struct {
	chunks           []LayerReadWriter
	activeChunkIndex int
	widthPerChunk    uint64
	lastChunkWidth   uint64
}

// A compile time check to ensure that GroupLayerReadWriter fully implements LayerReadWriter.
var _ LayerReadWriter = (*GroupLayerReadWriter)(nil)

// groupLayers groups a slice of layers into one unified layer.
func groupLayers(layers []LayerReadWriter) (*GroupLayerReadWriter, error) {
	if len(layers) < 2 {
		return nil, errors.New("number of layers must be at least 2")
	}

	widthPerLayer, err := layers[0].Width()
	if err != nil {
		return nil, err
	}
	if widthPerLayer == 0 {
		return nil, errors.New("0 width layers are not allowed")
	}

	// Verify that all layers, except the last one, have the same width.
	var lastLayerWidth uint64
	for i := 1; i < len(layers); i++ {
		layer := layers[i]
		if layer == nil {
			return nil, errors.New("nil layers are not allowed")
		}
		width, err := layers[i].Width()
		if err != nil {
			return nil, err
		}

		if i == len(layers)-1 {
			lastLayerWidth = width
		} else {
			if width != widthPerLayer && i < len(layers)-1 {
				return nil, errors.New("layers width mismatch")
			}
		}
	}

	g := &GroupLayerReadWriter{
		chunks:         layers,
		widthPerChunk:  widthPerLayer,
		lastChunkWidth: lastLayerWidth,
	}

	return g, nil
}

func (g *GroupLayerReadWriter) Seek(index uint64) error {
	// Find the target chunk.
	chunkIndex := int(index / g.widthPerChunk)
	if chunkIndex >= len(g.chunks) {
		return io.EOF
	}

	// If a new chunk was selected, reset all other chunks position.
	if chunkIndex != g.activeChunkIndex {
		for i, chunk := range g.chunks {
			if i == chunkIndex {
				continue
			}
			err := chunk.Seek(0)
			if err != nil {
				return err

			}
		}

		g.activeChunkIndex = chunkIndex
	}

	indexInChunk := index % g.widthPerChunk
	return g.chunks[g.activeChunkIndex].Seek(indexInChunk)
}

func (g *GroupLayerReadWriter) ReadNext() ([]byte, error) {
	val, err := g.chunks[g.activeChunkIndex].ReadNext()
	if err != nil {
		if err == io.EOF && g.activeChunkIndex < len(g.chunks)-1 {
			g.activeChunkIndex++
			return g.ReadNext()
		}
		return nil, err
	}

	return val, nil
}

func (g *GroupLayerReadWriter) Width() (uint64, error) {
	return uint64(len(g.chunks)-1)*g.widthPerChunk + g.lastChunkWidth, nil
}

func (g *GroupLayerReadWriter) Append(p []byte) (n int, err error) { panic("not implemented") }

func (g *GroupLayerReadWriter) Flush() error { panic("not implemented") }
