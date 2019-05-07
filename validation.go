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
	hash HashFunc) (bool, error) {
	valid, _, err := ValidatePartialTreeWithParkingSnapshots(leafIndices, leaves, proof, expectedRoot, hash)
	return valid, err
}

func ValidatePartialTreeWithParkingSnapshots(leafIndices []uint64, leaves, proof [][]byte, expectedRoot []byte,
	hash HashFunc) (bool, []ParkingSnapshot, error) {
	v, err := newValidator(leafIndices, leaves, proof, hash)
	if err != nil {
		return false, nil, err
	}
	root, parkingSnapshots, err := v.calcRoot(MaxUint)
	return bytes.Equal(root, expectedRoot), parkingSnapshots, err
}

func newValidator(leafIndices []uint64, leaves, proof [][]byte, hash HashFunc) (*validator, error) {
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
	if len(setOf(leafIndices...)) != len(leafIndices) {
		return nil, errors.New("leafIndices contain duplicates")
	}
	proofNodes := &proofIterator{proof}
	leafIt := &leafIterator{leafIndices, leaves}

	return &validator{leaves: leafIt, proofNodes: proofNodes, hash: hash}, nil
}

type validator struct {
	leaves     *leafIterator
	proofNodes *proofIterator
	hash       HashFunc
}

type ParkingSnapshot [][]byte

func (v *validator) calcRoot(stopAtLayer uint) ([]byte, []ParkingSnapshot, error) {
	activePos, activeNode, err := v.leaves.next()
	if err != nil {
		return nil, nil, err
	}
	var lChild, rChild, sibling []byte
	var subTreeSnapshots []ParkingSnapshot
	parkingSnapshots := []ParkingSnapshot{nil}
	for {
		if activePos.height == stopAtLayer {
			break
		}
		// The activeNode's sibling should be calculated iff it's an ancestor of the next proven leaf. Otherwise, the
		// sibling is the next node in the proof.
		nextLeafPos, _, err := v.leaves.peek()
		if err == nil && activePos.sibling().isAncestorOf(nextLeafPos) {
			sibling, subTreeSnapshots, err = v.calcRoot(activePos.height)
			if err != nil {
				return nil, nil, err
			}
		} else {
			sibling, err = v.proofNodes.next()
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
		activeNode = v.hash(lChild, rChild)
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
