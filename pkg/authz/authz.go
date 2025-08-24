package authz

import (
	"strings"

	"github.com/gofrs/uuid/v5"
)

type ID struct{ uuid.UUID }

type Principal interface {
	ID() ID
}

type user struct {
	id ID
}

func (u user) ID() ID {
	return u.id
}

func New(id ID) Principal {
	return user{id: id}
}

type Partial interface {
	SQL() (string, []any)
}

var NilPartial Partial = nilPartial{}

type nilPartial struct{}

func (p nilPartial) SQL() (string, []any) {
	return "", nil
}

type fileterPartial struct {
	filterBy []string
	values   []any
}

func (p fileterPartial) SQL() (string, []any) {
	if len(p.filterBy) == 0 {
		return "", nil
	}
	if len(p.filterBy) != len(p.values) {
		return "", nil
	}
	clauses := make([]string, 0, len(p.filterBy))
	args := make([]any, 0, len(p.values))
	for i, field := range p.filterBy {
		clauses = append(clauses, field+" = ?")
		args = append(args, p.values[i])
	}
	return "(" + strings.Join(clauses, " AND ") + ")", args
}

func FilterBy(key string, value any) fileterPartial {
	return fileterPartial{filterBy: []string{key}, values: []any{value}}
}

func (p fileterPartial) And(key string, value any) Partial {
	p.filterBy = append(p.filterBy, key)
	p.values = append(p.values, value)
	return p
}
