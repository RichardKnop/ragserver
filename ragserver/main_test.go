package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/weaviate/weaviate/entities/models"
)

func TestGetResults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		title       string
		given       *models.GraphQLResponse
		expected    []string
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
							map[string]any{"text": "foo"},
							map[string]any{"text": "bar"},
						},
					},
				},
			},
			[]string{"foo", "bar"},
			nil,
		},
	}

	for i, tst := range tests {
		t.Run(fmt.Sprintf("#%v_%v", i, tst.title), func(t *testing.T) {
			actual, err := decodeGetResults(tst.given)
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
