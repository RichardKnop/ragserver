package ragserver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFile_CompleteWithStatus(t *testing.T) {
	t.Parallel()

	updatedAt := time.Now().UTC()

	tests := []struct {
		name    string
		from    FileStatus
		to      FileStatus
		message string
		wantErr bool
	}{
		{
			name:    "processing to processed successfully",
			from:    FileStatusProcessing,
			to:      FileStatusProcessedSuccessfully,
			message: "",
			wantErr: false,
		},
		{
			name:    "processing to processing failed",
			from:    FileStatusProcessing,
			to:      FileStatusProcessingFailed,
			message: "some error message",
			wantErr: false,
		},
		{
			name:    "cannot change to processed successfully from non-processing status",
			from:    FileStatusUploaded,
			to:      FileStatusProcessedSuccessfully,
			message: "",
			wantErr: true,
		},
		{
			name:    "cannot change to processing failed from non-processing status",
			from:    FileStatusUploaded,
			to:      FileStatusProcessingFailed,
			message: "",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := &File{
				Status: tc.from,
			}
			err := f.CompleteWithStatus(tc.to, tc.message, updatedAt)
			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.to, f.Status)
			assert.Equal(t, tc.message, f.StatusMessage)
			assert.Equal(t, updatedAt, f.Updated)
		})
	}
}
