package merkle

import "github.com/spacemeshos/merkle-tree/cache"

type TreeBuilder struct {
	hash           HashFunc
	leavesToProves set
	cacheWriter    *cache.Writer
	minHeight      uint
}

func NewTreeBuilder() TreeBuilder {
	return TreeBuilder{}
}

func (tb TreeBuilder) Build() *Tree {
	if tb.hash == nil {
		tb.hash = GetSha256Parent
	}
	if tb.cacheWriter == nil {
		tb.cacheWriter = cache.NewWriter(cache.SpecificLayersPolicy(map[uint]bool{}), nil)
	}
	tb.cacheWriter.SetHash(tb.hash)
	return &Tree{
		baseLayer:     newLayer(0, tb.cacheWriter.GetLayerWriter(0)),
		hash:          tb.hash,
		leavesToProve: newSparseBoolStack(tb.leavesToProves),
		cacheWriter:   tb.cacheWriter,
		minHeight:     tb.minHeight,
	}
}

func (tb TreeBuilder) WithHashFunc(hash HashFunc) TreeBuilder {
	tb.hash = hash
	return tb
}

func (tb TreeBuilder) WithLeavesToProve(leavesToProves set) TreeBuilder {
	tb.leavesToProves = leavesToProves
	return tb
}

func (tb TreeBuilder) WithCacheWriter(cacheWriter *cache.Writer) TreeBuilder {
	tb.cacheWriter = cacheWriter
	return tb
}

func (tb TreeBuilder) WithMinHeight(minHeight uint) TreeBuilder {
	tb.minHeight = minHeight
	return tb
}

func NewTree() *Tree {
	return NewTreeBuilder().Build()
}

func NewProvingTree(leavesToProves set) *Tree {
	return NewTreeBuilder().WithLeavesToProve(leavesToProves).Build()
}

func NewCachingTree(cacheWriter *cache.Writer) *Tree {
	return NewTreeBuilder().WithCacheWriter(cacheWriter).Build()
}
