package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/RichardKnop/ragserver/pkg/authz"
)

func (a *Adapter) SavePrincipal(ctx context.Context, principal authz.Principal) error {
	if err := a.inTxDo(ctx, &sql.TxOptions{}, func(ctx context.Context, tx *sql.Tx) error {
		if err := execQuery(ctx, tx, insertPrincipalQuery{principal}); err != nil {
			return fmt.Errorf("exec query failed: %w", err)
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

type insertPrincipalQuery struct {
	authz.Principal
}

func (q insertPrincipalQuery) SQL() (string, []any) {
	query := `
		insert into "ragserver"."principal" (
			"id", 
			"name"
		)
		values (?, ?)
		on conflict("id") do update set
			"name"=excluded."name",
			"updated"=now()
	`
	args := []any{
		q.ID(),
		sql.NullString{String: q.Name(), Valid: q.Name() != ""},
	}

	return toPostgresParams(query), args
}
