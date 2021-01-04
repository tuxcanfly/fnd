package wire

import (
	"testing"
)

func TestBlobRes_Encoding(t *testing.T) {
	blobRes := &BlobRes{
		SectorSize:  1,
		Name:        "testname.",
		EpochHeight: 0,
		//PayloadPosition uint16,
		//PrevHash        crypto.Hash
		//MessageRoot     crypto.Hash
		//Signature       crypto.Signature
		Payload: nil,
	}

	testMessageEncoding(t, "blob_res", blobRes, &BlobRes{})
}
