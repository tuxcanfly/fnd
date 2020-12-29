package wire

import (
	"testing"

	"github.com/ddrp-org/ddrp/blob"
)

func TestSectorRes_Encoding(t *testing.T) {
	sectorRes := &SectorRes{
		Name:     "test",
		SectorID: 16,
		Sector:   blob.Sector{},
	}

	testMessageEncoding(t, "sector_res", sectorRes, &SectorRes{})
}
