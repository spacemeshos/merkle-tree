package readwriters

import (
	"io"

	"github.com/spacemeshos/merkle-tree/shared"
)

const NodeSize = shared.NodeSize

type SliceReadWriter struct {
	// a continuous memory for keeping nodes
	slice []byte
	// position in slice determined in nodes unit
	// must be multiplied by `NodeSize`` to get its
	// location in `slice`
	position uint64
}

// A compile time check to ensure that SliceReadWriter fully implements LayerReadWriter.
var _ shared.LayerReadWriter = (*SliceReadWriter)(nil)

func (s *SliceReadWriter) width() uint64 {
	return uint64(len(s.slice) / NodeSize)
}

func (s *SliceReadWriter) Width() (uint64, error) {
	return s.width(), nil
}

func (s *SliceReadWriter) Seek(index uint64) error {
	if index >= s.width() {
		return io.EOF
	}
	s.position = index
	return nil
}

func (s *SliceReadWriter) ReadNext() ([]byte, error) {
	if s.position >= s.width() {
		return nil, io.EOF
	}
	value := make([]byte, NodeSize)
	index := s.position * NodeSize
	copy(value, s.slice[index:index+NodeSize])
	s.position++
	return value, nil
}

func (s *SliceReadWriter) Append(p []byte) (n int, err error) {
	s.slice = append(s.slice, p...)
	return len(p), nil
}

func (s *SliceReadWriter) Flush() error {
	return nil
}

func (s *SliceReadWriter) Close() error {
	return nil
}
