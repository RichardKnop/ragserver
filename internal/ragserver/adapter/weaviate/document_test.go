package weaviate

import (
	"fmt"
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/weaviate/weaviate/entities/models"

	"github.com/RichardKnop/ragserver/internal/ragserver/core/ragserver"
)

func TestDecodeGetDocumentResults(t *testing.T) {
	t.Parallel()

	var (
		fileID1 = uuid.Must(uuid.FromString("9ea0b16a-7f4a-4a22-8ea1-ca2d932bafa8"))
		fileID2 = uuid.Must(uuid.FromString("1ad113d9-38f9-42d1-b205-4383250a4dfd"))
	)

	tests := []struct {
		title       string
		given       *models.GraphQLResponse
		expected    []ragserver.Document
		expectedErr error
	}{
		{
			"Missing Get key",
			&models.GraphQLResponse{
				Data: map[string]models.JSONObject{},
			},
			nil,
			fmt.Errorf("get key not found in result"),
		},
		{
			"Valid results",
			&models.GraphQLResponse{
				Data: map[string]models.JSONObject{
					"Get": map[string]any{
						"Document": []any{
							map[string]any{
								"text":    "foo",
								"file_id": fileID1.String(),
							},
							map[string]any{
								"text":    "bar",
								"file_id": fileID2.String(),
							},
						},
					},
				},
			},
			[]ragserver.Document{
				{
					Text:   "foo",
					FileID: ragserver.FileID{UUID: fileID1},
				},
				{
					Text:   "bar",
					FileID: ragserver.FileID{UUID: fileID2},
				},
			},
			nil,
		},
	}

	for i, tst := range tests {
		t.Run(fmt.Sprintf("#%v_%v", i, tst.title), func(t *testing.T) {
			actual, err := decodeGetDocumentResults(tst.given)
			if tst.expectedErr != nil {
				require.Error(t, err)
				assert.Equal(t, tst.expectedErr, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tst.expected, actual)
		})
	}
}
