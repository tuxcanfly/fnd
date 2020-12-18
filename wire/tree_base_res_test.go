package wire

import (
	"testing"

	"github.com/ddrp-org/ddrp/blob"
)

func TestTreeBaseRes_Encoding(t *testing.T) {
	treeBaseRes := &TreeBaseRes{
		Name:         "testname.",
		SectorHashes: blob.ZeroSectorHashes,
	}

	testMessageEncoding(t, "tree_base_res", treeBaseRes, &TreeBaseRes{})
}
