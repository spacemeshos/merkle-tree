package readwriters

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFileReadWriter(t *testing.T) {
	r := require.New(t)

	filename := "delete.me"
	readWriter, err := NewFileReadWriter(filename)
	r.NoError(err)

	defer func() {
		err = os.Remove(filename)
		r.NoError(err)
	}()

	n, err := readWriter.Append(makeLabel("something"))
	r.NoError(err)
	r.Equal(NodeSize, n)

	n, err = readWriter.Append(makeLabel("else"))
	r.NoError(err)
	r.Equal(NodeSize, n)

	err = readWriter.Flush()
	r.NoError(err)

	next, err := readWriter.ReadNext()
	r.NoError(err)
	r.Equal(string(makeLabel("something")), string(next))

	next, err = readWriter.ReadNext()
	r.NoError(err)
	r.Equal(string(makeLabel("else")), string(next))

	next, err = readWriter.ReadNext()
	r.EqualError(err, "EOF")
	r.Nil(next)

	err = readWriter.Seek(1)
	r.NoError(err)

	next, err = readWriter.ReadNext()
	r.NoError(err)
	r.Equal(string(makeLabel("else")), string(next))

	err = readWriter.Close()
	r.NoError(err)
}

func makeLabel(s string) []byte {
	return []byte(fmt.Sprintf("%32s", s))
}

func TestConsistentEOF(t *testing.T) {
	file, err := NewFileReadWriter(filepath.Join(t.TempDir(), "test"))
	require.NoError(t, err)
	slice := SliceReadWriter{}

	require.True(t, errors.Is(slice.Seek(1), io.EOF))
	require.True(t, errors.Is(file.Seek(1), io.EOF))
}
