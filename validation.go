package merkle

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
)

const MaxUint = ^uint(0)

// ValidatePartialTree uses leafIndices, leaves and proof to calculate the merkle root of the tree and then compares it
// to expectedRoot.
func ValidatePartialTree(leafIndices []uint64, leaves, proof [][]byte, expectedRoot []byte,
	hash HashFunc,
) (bool, error) {
	v, err := newValidator(leafIndices, leaves, proof, hash, false)
	if err != nil {
		return false, err
	}
	root, _, err := v.CalcRoot(MaxUint)
	return bytes.Equal(root, expectedRoot), err
}

// ValidatePartialTree uses leafIndices, leaves and proof to calculate the merkle root of the tree and then compares it
// to expectedRoot. Additionally, it reconstructs the parked nodes when each proven leaf was originally added to the
// tree and returns a list of snapshots. This method is ~15% slower than ValidatePartialTree.
func ValidatePartialTreeWithParkingSnapshots(leafIndices []uint64, leaves, proof [][]byte, expectedRoot []byte,
	hash HashFunc,
) (bool, []ParkingSnapshot, error) {
	v, err := newValidator(leafIndices, leaves, proof, hash, true)
	if err != nil {
		return false, nil, err
	}
	root, parkingSnapshots, err := v.CalcRoot(MaxUint)
	return bytes.Equal(root, expectedRoot), parkingSnapshots, err
}

func newValidator(leafIndices []uint64, leaves, proof [][]byte, hash HashFunc, storeSnapshots bool) (*Validator, error) {
	if len(leafIndices) != len(leaves) {
		return nil, fmt.Errorf("number of leaves (%d) must equal number of indices (%d)", len(leaves),
			len(leafIndices))
	}
	if len(leaves) == 0 {
		return nil, errors.New("at least one leaf is required for validation")
	}
	if !sort.SliceIsSorted(leafIndices, func(i, j int) bool { return leafIndices[i] < leafIndices[j] }) {
		return nil, errors.New("leafIndices are not sorted")
	}
	if len(SetOf(leafIndices...)) != len(leafIndices) {
		return nil, errors.New("leafIndices contain duplicates")
	}
	proofNodes := &proofIterator{proof}
	leafIt := &LeafIterator{leafIndices, leaves}

	return &Validator{Leaves: leafIt, ProofNodes: proofNodes, Hash: hash, StoreSnapshots: storeSnapshots}, nil
}

type Validator struct {
	Leaves         *LeafIterator
	ProofNodes     *proofIterator
	Hash           HashFunc
	StoreSnapshots bool
}

type ParkingSnapshot [][]byte

func (v *Validator) CalcRoot(stopAtLayer uint) ([]byte, []ParkingSnapshot, error) {
	activePos, activeNode, err := v.Leaves.next()
	if err != nil {
		return nil, nil, err
	}
	var lChild, rChild, sibling []byte
	var parkingSnapshots, subTreeSnapshots []ParkingSnapshot
	if v.StoreSnapshots {
		parkingSnapshots = []ParkingSnapshot{nil}
	}
	for {
		if activePos.Height == stopAtLayer {
			break
		}
		// The activeNode's sibling should be calculated iff it's an ancestor of the next proven leaf. Otherwise, the
		// sibling is the next node in the proof.
		nextLeafPos, _, err := v.Leaves.peek()
		if err == nil && activePos.sibling().isAncestorOf(nextLeafPos) {
			sibling, subTreeSnapshots, err = v.CalcRoot(activePos.Height)
			if err != nil {
				return nil, nil, err
			}
		} else {
			sibling, err = v.ProofNodes.next()
			if err == noMoreItems {
				break
			}
		}
		if activePos.isRightSibling() {
			lChild, rChild = sibling, activeNode
			addToAll(parkingSnapshots, lChild)
		} else {
			lChild, rChild = activeNode, sibling
			addToAll(parkingSnapshots, nil)
			if len(subTreeSnapshots) > 0 {
				parkingSnapshots = append(parkingSnapshots, addToAll(subTreeSnapshots, activeNode)...)
				subTreeSnapshots = nil
			}
		}
		activeNode = v.Hash(nil, lChild, rChild)
		activePos = activePos.parent()
	}
	return activeNode, parkingSnapshots, nil
}

func addToAll(snapshots []ParkingSnapshot, node []byte) []ParkingSnapshot {
	for i := 0; i < len(snapshots); i++ {
		snapshots[i] = append(snapshots[i], node)
	}
	return snapshots
}
