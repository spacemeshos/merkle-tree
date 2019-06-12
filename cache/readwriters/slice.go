package readwriters

import (
	"github.com/spacemeshos/merkle-tree/shared"
	"io"
)

const NodeSize = shared.NodeSize

type SliceReadWriter struct {
	slice    [][]byte
	position uint64
}

// A compile time check to ensure that SliceReadWriter fully implements LayerReadWriter.
var _ shared.LayerReadWriter = (*SliceReadWriter)(nil)

func (s *SliceReadWriter) Width() (uint64, error) {
	return uint64(len(s.slice)), nil
}

func (s *SliceReadWriter) Seek(index uint64) error {
	if index >= uint64(len(s.slice)) {
		return io.EOF
	}
	s.position = index
	return nil
}

func (s *SliceReadWriter) ReadNext() ([]byte, error) {
	if s.position >= uint64(len(s.slice)) {
		return nil, io.EOF
	}
	value := make([]byte, NodeSize)
	copy(value, s.slice[s.position])
	s.position++
	return value, nil
}

func (s *SliceReadWriter) Append(p []byte) (n int, err error) {
	s.slice = append(s.slice, p)
	return len(p), nil
}

func (s *SliceReadWriter) Flush() error {
	return nil
}

func (s *SliceReadWriter) Close() error {
	return nil
}
