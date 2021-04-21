package wire

import (
	"testing"
)

func TestUpdateReq_Encoding(t *testing.T) {
	updateReq := &BlobUpdateReq{
		Name:        "testname",
		EpochHeight: 0,
		SectorSize:  0,
	}

	testMessageEncoding(t, "update_req", updateReq, &BlobUpdateReq{})
}
