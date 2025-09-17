package ragserver

import (
	"fmt"
	"strings"
)

type SortOrder string

const (
	SortOrderAsc  SortOrder = "ASC"
	SortOrderDesc SortOrder = "DESC"
)

type SortParams struct {
	Limit int
	By    string
	Order SortOrder
}

func (p SortParams) Empty() bool {
	return p.Limit == 0 && p.By == "" && p.Order == ""
}

func (p SortParams) Valid(sortableBy []string) bool {
	if p.Limit < 0 {
		return false
	}

	if p.By != "" {
		var found bool
		for _, s := range sortableBy {
			if s == p.By {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func (p SortParams) SQL() string {
	var s string

	if p.By != "" {
		s += fmt.Sprintf(" order by %s", p.By)
		if p.Order != "" {
			s += fmt.Sprintf(" %s", strings.ToLower(string(p.Order)))
		}
	}

	if p.Limit > 0 {
		s += fmt.Sprintf(" limit %d", p.Limit)
	}

	return s
}
