package p2p

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLookupDNSSeeds(t *testing.T) {
	seeds, err := ResolveDNSSeeds("seeds-test.ddrp.network")
	require.NoError(t, err)

	require.Equal(t, 4, len(seeds))
	require.Contains(t, seeds, "198.54.117.200")
	require.Contains(t, seeds, "198.54.117.199")
	require.Contains(t, seeds, "198.54.117.198")
	require.Contains(t, seeds, "198.54.117.197")
}
