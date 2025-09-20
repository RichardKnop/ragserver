package weaviate

import (
	"fmt"
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/weaviate/weaviate/entities/models"

	"github.com/RichardKnop/ragserver"
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
								"content": "foo",
								"page":    float64(5),
								"file_id": fileID1.String(),
							},
							map[string]any{
								"content": "bar",
								"page":    float64(43),
								"file_id": fileID2.String(),
							},
						},
					},
				},
			},
			[]ragserver.Document{
				{
					Content: "foo",
					Page:    5,
					FileID:  ragserver.FileID{UUID: fileID1},
				},
				{
					Content: "bar",
					Page:    43,
					FileID:  ragserver.FileID{UUID: fileID2},
				},
			},
			nil,
		},
	}

	for i, tc := range tests {
		t.Run(fmt.Sprintf("#%v_%v", i, tc.title), func(t *testing.T) {
			actual, err := decodeGetDocumentResults(tc.given)
			if tc.expectedErr != nil {
				require.Error(t, err)
				assert.Equal(t, tc.expectedErr, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
