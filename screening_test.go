package ragserver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScreening_CompleteWithStatus(t *testing.T) {
	t.Parallel()

	updatedAt := time.Now().UTC()

	tests := []struct {
		name    string
		from    ScreeningStatus
		to      ScreeningStatus
		message string
		wantErr bool
	}{
		{
			name:    "generating to completed",
			from:    ScreeningStatusGenerating,
			to:      ScreeningStatusCompleted,
			message: "",
			wantErr: false,
		},
		{
			name:    "generating to failed",
			from:    ScreeningStatusGenerating,
			to:      ScreeningStatusFailed,
			message: "some error message",
			wantErr: false,
		},
		{
			name:    "cannot change to completed from non-generating status",
			from:    ScreeningStatusRequested,
			to:      ScreeningStatusCompleted,
			message: "",
			wantErr: true,
		},
		{
			name:    "cannot change to failed from non-generating status",
			from:    ScreeningStatusRequested,
			to:      ScreeningStatusFailed,
			message: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &Screening{
				Status: tt.from,
			}
			err := s.CompleteWithStatus(tt.to, tt.message, updatedAt)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.to, s.Status)
			assert.Equal(t, tt.message, s.StatusMessage)
			assert.Equal(t, updatedAt, s.Updated)
		})
	}
}
