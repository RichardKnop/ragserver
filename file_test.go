package ragserver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFile_ChangeStatus(t *testing.T) {
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
			name:    "empty status to uploaded",
			from:    "",
			to:      FileStatusUploaded,
			message: "",
			wantErr: false,
		},
		{
			name:    "uploaded to processing",
			from:    FileStatusUploaded,
			to:      FileStatusProcessing,
			message: "",
			wantErr: false,
		},
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
			name:    "cannot change to processing from non-uploaded status",
			from:    FileStatusProcessingFailed,
			to:      FileStatusProcessing,
			message: "",
			wantErr: true,
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

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := &File{
				Status: tt.from,
			}
			err := f.ChangeStatus(tt.to, tt.message, updatedAt)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.to, f.Status)
			assert.Equal(t, tt.message, f.StatusMessage)
			assert.Equal(t, updatedAt, f.UpdatedAt.T)
		})
	}
}
