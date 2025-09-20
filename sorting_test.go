package ragserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSortParams_Valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		params     SortParams
		sortableBy []string
		valid      bool
	}{
		{
			"negative limit is invalid",
			SortParams{
				Limit: -1,
			},
			nil,
			false,
		},
		{
			"cannot sort by non-sortable field",
			SortParams{
				By: "bogus",
			},
			[]string{"foo", "bar"},
			false,
		},
		{
			"valid sort params",
			SortParams{
				By:    "foo",
				Order: SortOrderDesc,
			},
			[]string{"foo", "bar"},
			true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			valid := tc.params.Valid(tc.sortableBy)
			assert.Equal(t, tc.valid, valid)
		})
	}
}

func TestSortParams_SQL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		params   SortParams
		expected string
	}{
		{
			"empty",
			SortParams{},
			"",
		},
		{
			"only limit",
			SortParams{
				Limit: 10,
			},
			" limit 10",
		},
		{
			"sort by without order",
			SortParams{
				By: "foo",
			},
			" order by foo",
		},
		{
			"sort by with asc order",
			SortParams{
				By:    "foo",
				Order: SortOrderAsc,
			},
			" order by foo asc",
		},
		{
			"sort by with desc order",
			SortParams{
				By:    "foo",
				Order: SortOrderDesc,
			},
			" order by foo desc",
		},
		{
			"sort by with limit",
			SortParams{
				By:    "foo",
				Order: SortOrderDesc,
				Limit: 10,
			},
			" order by foo desc limit 10",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.params.SQL()
			assert.Equal(t, tc.expected, actual)
		})
	}
}
