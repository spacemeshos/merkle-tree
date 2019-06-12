package readwriters

import (
	"bufio"
	"fmt"
	"github.com/spacemeshos/merkle-tree/shared"
	"io"
	"os"
)

const OwnerReadWrite = 0600

func NewFileReadWriter(filename string) (*FileReadWriter, error) {
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, OwnerReadWrite)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for disk read-writer: %v", err)
	}
	return &FileReadWriter{
		f: f,
		b: bufio.NewReadWriter(bufio.NewReader(f), bufio.NewWriter(f)),
	}, nil
}

type FileReadWriter struct {
	f *os.File
	b *bufio.ReadWriter
}

// A compile time check to ensure that FileReadWriter fully implements LayerReadWriter.
var _ shared.LayerReadWriter = (*FileReadWriter)(nil)

func (rw *FileReadWriter) Seek(index uint64) error {
	_, err := rw.f.Seek(int64(index*NodeSize), io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to seek in disk reader: %v", err)
	}
	rw.b.Reader.Reset(rw.f)
	return err
}

func (rw *FileReadWriter) ReadNext() ([]byte, error) {
	ret := make([]byte, NodeSize)
	_, err := rw.b.Read(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (rw *FileReadWriter) Width() (uint64, error) {
	info, err := rw.f.Stat()
	if err != nil {
		return 0, fmt.Errorf("failed to get stats for disk reader: %v", err)
	}
	return uint64(info.Size()) / NodeSize, nil
}

func (rw *FileReadWriter) Append(p []byte) (n int, err error) {
	n, err = rw.b.Write(p)
	return
}

func (rw *FileReadWriter) Flush() error {
	err := rw.b.Flush()
	if err != nil {
		return fmt.Errorf("failed to flush disk writer: %v", err)
	}
	err = rw.Seek(0)
	if err != nil {
		return fmt.Errorf("failed to seek disk reader to start of file: %v", err)
	}
	return nil
}

func (rw *FileReadWriter) Close() error {
	err := rw.b.Flush()
	if err != nil {
		return fmt.Errorf("failed to flush disk writer: %v", err)
	}
	rw.b = nil

	err = rw.f.Close()
	if err != nil {
		return err
	}
	rw.f = nil

	return nil
}
