package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_toPostgresParams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		sql      string
		expected string
	}{
		{
			"empty",
			"",
			"",
		},
		{
			"insert query",
			"INSERT INTO foo (x, y) VALUES (?, ?)",
			"INSERT INTO foo (x, y) VALUES ($1, $2)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := toPostgresParams(tt.sql)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
