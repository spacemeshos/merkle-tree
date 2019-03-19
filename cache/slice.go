package cache

import "io"

type SliceReadWriter struct {
	slice    [][]byte
	position uint64
}

func (s *SliceReadWriter) Width() uint64 {
	return uint64(len(s.slice))
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

func (s *SliceReadWriter) Write(p []byte) (n int, err error) {
	s.slice = append(s.slice, p)
	return len(p), nil
}
