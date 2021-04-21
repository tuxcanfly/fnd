package wire

import "testing"

func TestNilUpdate_Encoding(t *testing.T) {
	nilUpdate := NewBlobNilUpdate("testname.")
	testMessageEncoding(t, "nil_update", nilUpdate, &BlobNilUpdate{})
}
