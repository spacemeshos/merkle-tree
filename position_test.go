package merkle

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPosition_isAncestorOf(t *testing.T) {
	lower := position{
		index:  0,
		height: 0,
	}

	higher := position{
		index:  0,
		height: 1,
	}

	isAncestor := lower.isAncestorOf(higher)

	require.False(t, isAncestor)
}
