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

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := toPostgresParams(tc.sql)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
