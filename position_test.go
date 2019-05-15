package merkle

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPosition_isAncestorOf(t *testing.T) {
	lower := Position{
		Index:  0,
		Height: 0,
	}

	higher := Position{
		Index:  0,
		Height: 1,
	}

	isAncestor := lower.isAncestorOf(higher)

	require.False(t, isAncestor)
}
