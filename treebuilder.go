package merkle

import "github.com/spacemeshos/merkle-tree/cache"

type TreeBuilder struct {
	hash           HashFunc
	leavesToProves []uint64
	cache          *cache.Cache
	minHeight      uint
}

func NewTreeBuilder() TreeBuilder {
	return TreeBuilder{}
}

func (tb TreeBuilder) Build() *Tree {
	if tb.hash == nil {
		tb.hash = GetSha256Parent
	}
	if tb.cache == nil {
		tb.cache = &cache.Cache{}
	}
	tb.cache.Hash = tb.hash
	return &Tree{
		baseLayer:     newLayer(0, tb.cache.GetLayerWriter(0)),
		hash:          tb.hash,
		leavesToProve: newSparseBoolStack(tb.leavesToProves),
		cache:         tb.cache,
		minHeight:     tb.minHeight,
	}
}

func (tb TreeBuilder) WithHashFunc(hash HashFunc) TreeBuilder {
	tb.hash = hash
	return tb
}

func (tb TreeBuilder) WithLeavesToProve(leavesToProves []uint64) TreeBuilder {
	tb.leavesToProves = leavesToProves
	return tb
}

func (tb TreeBuilder) WithCache(treeCache *cache.Cache) TreeBuilder {
	tb.cache = treeCache
	return tb
}

func (tb TreeBuilder) WithMinHeight(minHeight uint) TreeBuilder {
	tb.minHeight = minHeight
	return tb
}

func NewTree() *Tree {
	return NewTreeBuilder().Build()
}

func NewProvingTree(leavesToProves []uint64) *Tree {
	return NewTreeBuilder().WithLeavesToProve(leavesToProves).Build()
}

func NewCachingTree(treeCache *cache.Cache) *Tree {
	return NewTreeBuilder().WithCache(treeCache).Build()
}
