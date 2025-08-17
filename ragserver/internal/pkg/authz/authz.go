package authz

import (
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
