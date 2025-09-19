package store

import (
	"github.com/gofrs/uuid/v5"

	"github.com/RichardKnop/ragserver/pkg/authz"
)

var testPrincipal = authz.New(
	authz.ID{UUID: uuid.Must(uuid.NewV4())},
	"test principal",
)

func (s *StoreTestSuite) TestSavePrincipal() {
	ctx, cancel := testContext()
	defer cancel()

	s.Require().NoError(s.adapter.SavePrincipal(ctx, testPrincipal), "error saving principal")
	s.Require().NoError(s.adapter.SavePrincipal(ctx, testPrincipal), "error saving principal again (upsert)")

	// Check principal was saved
	stmt, err := s.db.Prepare(toPostgresParams(`select "id", "name" from "principal" where "id" = ?`))
	s.Require().NoError(err)
	defer stmt.Close()

	var (
		id   authz.ID
		name string
	)
	err = stmt.QueryRowContext(ctx, testPrincipal.ID()).Scan(&id, &name)
	s.Require().NoError(err)
	s.Equal(testPrincipal.ID(), id)
	s.Equal(testPrincipal.Name(), name)
}
