package shared

type HashFunc func(buf, lChild, rChild []byte) []byte

// LayerReadWriter is a combined reader-writer. Note that the Seek() method only belongs to the LayerReader interface
// and does not affect the LayerWriter.
type LayerReadWriter interface {
	Seek(index uint64) error
	ReadNext() ([]byte, error)
	Width() (uint64, error)
	Append(p []byte) (n int, err error)
	Flush() error
	Close() error
}

type LayerReader interface {
	Seek(index uint64) error
	ReadNext() ([]byte, error)
	Width() (uint64, error)
	Close() error
}

type LayerWriter interface {
	Append(p []byte) (n int, err error)
	Flush() error
	Close() error
}

type CacheWriter interface {
	SetLayer(layerHeight uint, rw LayerReadWriter)
	GetLayerWriter(layerHeight uint) (LayerWriter, error)
	SetHash(hashFunc HashFunc)
	GetReader() (CacheReader, error)
}

type CacheReader interface {
	Layers() map[uint]LayerReadWriter
	GetLayerReader(layerHeight uint) LayerReader
	GetHashFunc() HashFunc
	GetLayerFactory() LayerFactory
	GetCachingPolicy() CachingPolicy
}

type CachingPolicy func(layerHeight uint) (shouldCacheLayer bool)

type LayerFactory func(layerHeight uint) (LayerReadWriter, error)
