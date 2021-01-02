package protocol

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCheckTimebank(t *testing.T) {
	params := &TimebankParams{
		TimebankDuration:     48 * time.Hour,
		MinUpdateInterval:    2 * time.Minute,
		FullUpdatesPerPeriod: 2,
	}

	now := time.Now()
	tests := []struct {
		name           string
		prevUpdateTime time.Time
		prevTimebank   int
		sectorsUpdated int
		newTimebank    int
	}{
		{
			"zero bytes updated",
			now,
			0,
			0,
			-1,
		},
		{
			"more than sector count updated",
			now,
			0,
			256,
			-1,
		},
		{
			"update is within min update frequency",
			now,
			0,
			1,
			-1,
		},
		{
			"not enough time bank - one sector interval",
			now.Add(-1 * 10 * time.Minute),
			0,
			32,
			-1,
		},
		{
			"not enough time bank - multiple sector intervals",
			now.Add(-1 * 24 * time.Hour),
			0,
			4097,
			-1,
		},
		{
			"enough time bank - one sector interval",
			now.Add(-1 * 10 * time.Minute),
			2,
			1,
			29,
		},
		{
			"enough time bank - multiple sector intervals",
			now.Add(-1 * 24 * time.Hour),
			0,
			100,
			4014,
		},
		{
			"enough time bank - multiple sector intervals with high initial bank",
			now.Add(-1 * 24 * time.Hour),
			512,
			100,
			4526,
		},
		{
			"enough time bank - multiple sector intervals zero bank",
			now.Add(-1 * params.TimebankDuration),
			0,
			100,
			8092,
		},
		{
			"enough time bank - no previous update time",
			time.Time{},
			0,
			32,
			8160,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.newTimebank, CheckTimebank(params, tt.prevUpdateTime, tt.prevTimebank, tt.sectorsUpdated))
		})
	}
}
