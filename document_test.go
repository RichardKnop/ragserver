package ragserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchSnippetsToDocuments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                      string
		snippets                  []string
		documents                 []Document
		expectedMatchedDocuments  []Document
		expectedUnmatchedSnippets []string
	}{
		{
			"no matches",
			[]string{
				"nonexistent1",
				"nonexistent2",
			},
			[]Document{
				{Content: "snippet1"},
				{Content: "snippet2"},
			},
			nil,
			[]string{
				"nonexistent1",
				"nonexistent2",
			},
		},
		{
			"all matching",
			[]string{
				"snippet1",
				"snippet2",
			},
			[]Document{
				{Content: "snippet1"},
				{Content: "snippet2"},
			},
			[]Document{
				{Content: "snippet1"},
				{Content: "snippet2"},
			},
			nil,
		},
		{
			"some matching",
			[]string{
				"snippet1",
				"nonexistent2",
			},
			[]Document{
				{Content: "snippet1"},
				{Content: "snippet2"},
			},
			[]Document{
				{Content: "snippet1"},
			},
			[]string{
				"nonexistent2",
			},
		},
		{
			"partial matches",
			[]string{
				"(7) We set our target using the International Energy Agency Net-Zero Emissions by 2050 scenario.",
				"nonexistent2",
				"Achieve net-zero GHG emissions by 2050, including operational emissions (Scope 1 and 2) and emissions attributable to our financing (Scope 3, Category 15).",
			},
			[]Document{
				{Content: "snippet1"},
				{Content: "(6) Metric tons of CO2 per metric ton of steel (7) We set our target using the International Energy Agency Net-Zero Emissions by 2050 scenario."},
				{Content: "snippet2"},
				{Content: "• Achieve net-zero GHG emissions by 2050, including operational emissions (Scope 1 and 2) and emissions attributable to our financing (Scope 3, Category 15)."},
			},
			[]Document{
				{Content: "(6) Metric tons of CO2 per metric ton of steel (7) We set our target using the International Energy Agency Net-Zero Emissions by 2050 scenario."},
				{Content: "• Achieve net-zero GHG emissions by 2050, including operational emissions (Scope 1 and 2) and emissions attributable to our financing (Scope 3, Category 15)."},
			},
			[]string{
				"nonexistent2",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			matchedDocuments, unmatchedSnippets := MatchSnippetsToDocuments(tc.snippets, tc.documents)
			assert.Equal(t, tc.expectedMatchedDocuments, matchedDocuments)
			assert.Equal(t, tc.expectedUnmatchedSnippets, unmatchedSnippets)
		})
	}
}
