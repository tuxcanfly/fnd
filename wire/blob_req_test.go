package wire

import (
	"testing"
)

func TestBlobReq_Encoding(t *testing.T) {
	sectorReq := &BlobReq{
		Name:        "testname.",
		EpochHeight: 0,
		SectorSize:  1,
	}

	testMessageEncoding(t, "sector_req", sectorReq, &BlobReq{})
}
