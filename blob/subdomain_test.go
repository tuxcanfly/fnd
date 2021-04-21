package blob

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/btcsuite/btcd/btcec"
	"github.com/stretchr/testify/require"
)

var (
	fixedEpochHeight = uint16(0)
	fixedSize        = uint8(0)
)

func TestSubdomain_Encoding(t *testing.T) {
	keyHex := "02ce4b1cf077d919934e02efd281569ad9da306d229e707c6f36572bfc645ee694"
	keyBytes, err := hex.DecodeString(keyHex)
	key, err := btcec.ParsePubKey(keyBytes, btcec.S256())
	if err != nil {
		panic(err)
	}

	fixture := Subdomain{
		ID:          1,
		Name:        "hello",
		EpochHeight: fixedEpochHeight,
		Size:        fixedSize,
		PublicKey:   key,
	}

	var buf bytes.Buffer
	err = fixture.Encode(&buf)
	require.NoError(t, err)

	r := bufio.NewReader(&buf)
	decoded := Subdomain{}
	err = decoded.Decode(r)
	require.NoError(t, err)

	require.Equal(t, fixture.ID, decoded.ID)
	require.Equal(t, fixture.Name, decoded.Name)
	require.Equal(t, fixture.EpochHeight, decoded.EpochHeight)
	require.Equal(t, fixture.Size, decoded.Size)
	require.Equal(t, fixture.PublicKey, decoded.PublicKey)
}
