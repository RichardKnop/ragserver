package pdf

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neurosnap/sentences"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RichardKnop/ragserver"
)

func TestExtract(t *testing.T) {
	t.Parallel()

	items := []item{
		{
			PageNumber: 3,
			Text:       "foo",
			Type:       "Text",
		},
		{
			PageNumber: 5,
			Text:       "bar",
			Type:       "List item",
		},
	}

	training, err := sentences.LoadTraining([]byte(ragserver.TestEn))
	require.NoError(t, err)

	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/html" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("<html></html>"))
		} else {
			data, _ := json.Marshal(items)
			w.WriteHeader(http.StatusOK)
			w.Write(data)
		}
	}))
	defer svr.Close()

	adapter := New(training, WithBaseURL(svr.URL))

	t.Run("Without relevant topics", func(t *testing.T) {
		documents, err := adapter.Extract(context.Background(), "test.pdf", bytes.NewReader([]byte("test")), ragserver.RelevantTopics{})
		require.NoError(t, err)

		expected := []ragserver.Document{
			{
				Content: "foo",
				Page:    3,
			},
			{
				Content: "bar",
				Page:    5,
			},
		}
		assert.Equal(t, expected, documents)
	})

	t.Run("With relevant topics", func(t *testing.T) {
		documents, err := adapter.Extract(context.Background(), "test.pdf", bytes.NewReader([]byte("test")), ragserver.RelevantTopics{
			{Keywords: []string{"foo"}},
		})
		require.NoError(t, err)

		expected := []ragserver.Document{
			{
				Content: "foo",
				Page:    3,
			},
		}
		assert.Equal(t, expected, documents)
	})
}
