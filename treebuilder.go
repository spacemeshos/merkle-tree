package merkle

type TreeBuilder struct {
	hash           HashFunc
	leavesToProves Set
	cacheWriter    CacheWriter
	minHeight      uint
}

func NewTreeBuilder() TreeBuilder {
	return TreeBuilder{}
}

func (tb TreeBuilder) Build() (*Tree, error) {
	if tb.hash == nil {
		tb.hash = GetSha256Parent
	}
	if tb.cacheWriter == nil {
		tb.cacheWriter = disabledCacheWriter{}
	}
	tb.cacheWriter.SetHash(tb.hash)
	writer, err := tb.cacheWriter.GetLayerWriter(0)
	if err != nil {
		return &Tree{}, err
	}
	return &Tree{
		baseLayer:     newLayer(0, writer),
		hash:          tb.hash,
		leavesToProve: NewSparseBoolStack(tb.leavesToProves),
		cacheWriter:   tb.cacheWriter,
		minHeight:     tb.minHeight,
	}, nil
}

func (tb TreeBuilder) WithHashFunc(hash HashFunc) TreeBuilder {
	tb.hash = hash
	return tb
}

func (tb TreeBuilder) WithLeavesToProve(leavesToProves map[uint64]bool) TreeBuilder {
	tb.leavesToProves = leavesToProves
	return tb
}

func (tb TreeBuilder) WithCacheWriter(cacheWriter CacheWriter) TreeBuilder {
	tb.cacheWriter = cacheWriter
	return tb
}

func (tb TreeBuilder) WithMinHeight(minHeight uint) TreeBuilder {
	tb.minHeight = minHeight
	return tb
}

func NewTree() (*Tree, error) {
	return NewTreeBuilder().Build()
}

func NewProvingTree(leavesToProves map[uint64]bool) (*Tree, error) {
	return NewTreeBuilder().WithLeavesToProve(leavesToProves).Build()
}

func NewCachingTree(cacheWriter CacheWriter) (*Tree, error) {
	return NewTreeBuilder().WithCacheWriter(cacheWriter).Build()
}

type disabledCacheWriter struct{}

// A compile time check to ensure that disabledCacheWriter fully implements CacheWriter.
var _ CacheWriter = (*disabledCacheWriter)(nil)

func (disabledCacheWriter) SetLayer(layerHeight uint, rw LayerReadWriter)        {}
func (disabledCacheWriter) GetLayerWriter(layerHeight uint) (LayerWriter, error) { return nil, nil }
func (disabledCacheWriter) SetHash(hashFunc HashFunc)                            {}
func (disabledCacheWriter) GetReader() (CacheReader, error)                      { return nil, nil }
