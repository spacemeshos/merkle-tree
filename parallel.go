package merkle

import (
	"sync"
	"sync/atomic"
)

// maxConcurrency sets the max number of threads that can update the tree at the same time.
const maxConcurrency = 32

// ParallelTree calculates a merkle tree root. Leaves are appended to the tree incrementally. It is thread safe and
// performs tree construction concurrently. Concurrency can be controlled with the maxConcurrency constant.
//
// It has the following methods:
//
// 	AddLeaf(leaf Node)
// AddLeaf updates the state of the tree with another leaf.
//
//	Root() (Node, error)
// Root returns the root of the tree or an error if the number of leaves added is not a power of 2.
type ParallelTree struct {
	*sync.Mutex // protects the layers slice from concurrent read/writes
	leafCount uint64
	getParent func(leftChild, rightChild Node) Node
	layers    []layer
	wg        sync.WaitGroup // ensures that leaf-adding threads are complete before returning the root
	guard     chan struct{}  // limit concurrency
}

// NewParallelTree creates an empty tree structure that leaves can be added to. When all leaves have been added the root
// can be queried.
func NewParallelTree(getParent func(leftChild, rightChild Node) Node) ParallelTree {
	return ParallelTree{
		Mutex:     &sync.Mutex{},
		getParent: getParent,
		guard:     make(chan struct{}, maxConcurrency),
	}
}

// AddLeaf incorporates a new leaf to the state of the tree. It updates the state required to eventually determine the
// root of the tree.
func (t *ParallelTree) AddLeaf(leaf Node) {
	index := atomic.AddUint64(&t.leafCount, 1) - 1
	t.wg.Add(1)
	t.guard <- struct{}{}
	go t.addNode(0, index, leaf)
}

func (t *ParallelTree) addNode(height int, index uint64, node Node) {
	t.ensureHeight(height)
	if sibling, found := t.layers[height].popSiblingOrAddNode(index, node); found {
		if index%2 == 0 {
			node = t.getParent(node, sibling)
		} else {
			node = t.getParent(sibling, node)
		}
		t.addNode(height+1, index>>1, node)
	} else {
		t.wg.Done()
		<-t.guard
	}
}

// ensureHeight checks if the tree has the specified layer. If it doesn't, layers are added until it does.
func (t *ParallelTree) ensureHeight(height int) {
	if len(t.layers) > height {
		return
	}
	t.Lock()
	for len(t.layers) <= height {
		t.layers = append(t.layers, layer{
			pendingNodes: make(map[uint64]Node),
			Mutex:        &sync.Mutex{},
		})
	}
	t.Unlock()
}

func (t *ParallelTree) Root() (Node, error) {
	t.wg.Wait()
	if !t.isFull() {
		return nil, ErrorIncompleteTree
	}
	root := t.layers[len(t.layers)-1].pendingNodes[0]
	return root, nil
}

func (t *ParallelTree) isFull() bool {
	for height := 0; ; height++ {
		if height == len(t.layers)-1 {
			return len(t.layers[height].pendingNodes) == 1
		}
		if len(t.layers[height].pendingNodes) != 0 {
			return false
		}
	}
}

// layer stores pending nodes
type layer struct {
	*sync.Mutex // protects the pendingNodes map from concurrent read/writes
	pendingNodes map[uint64]Node
}

func (l *layer) popSiblingOrAddNode(index uint64, node Node) (Node, bool) {
	siblingIndex := index ^ 1
	l.Lock()
	defer l.Unlock()
	if sibling, found := l.pendingNodes[siblingIndex]; found {
		delete(l.pendingNodes, siblingIndex)
		return sibling, true
	}
	l.pendingNodes[index] = node
	return nil, false
}
