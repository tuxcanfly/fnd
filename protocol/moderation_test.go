package protocol

import (
	"fnd/blob"
	"fnd/store"
	"fnd/testutil/testfs"
	"testing"

	"github.com/stretchr/testify/require"
)

// TODO: refactor to use the httptest server

func TestIngestBanLists(t *testing.T) {
	db, doneDB := setupDB(t)
	defer doneDB()
	tmpDir, doneDir := testfs.NewTempDir(t)
	defer doneDir()

	bs := blob.NewStore(tmpDir)
	bl, err := bs.Open("foo", blob.Size)
	require.NoError(t, err)
	require.NoError(t, bl.Close())

	err = IngestBanLists(db, bs, []string{
		"http://sprunge.us/fKKV87",
		"http://sprunge.us/Ot8MYp",
	})
	require.NoError(t, err)

	fooBanned, err := store.NameIsBanned(db, "foo")
	require.NoError(t, err)
	require.True(t, fooBanned)
	barBanned, err := store.NameIsBanned(db, "bar")
	require.NoError(t, err)
	require.True(t, barBanned)
	bazBanned, err := store.NameIsBanned(db, "baz")
	require.NoError(t, err)
	require.False(t, bazBanned)

	exists, err := bs.Exists("foo")
	require.NoError(t, err)
	require.False(t, exists)
}
