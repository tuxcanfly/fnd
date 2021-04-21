package wire

import (
	"testing"
)

func TestUpdate_Encoding(t *testing.T) {
	update := &BlobUpdate{
		Name:        "testname",
		EpochHeight: fixedEpochHeight,
		SectorSize:  fixedSectorSize,
	}

	testMessageEncoding(t, "update", update, &BlobUpdate{})
}
