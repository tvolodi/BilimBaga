package deptscope

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

// Store answers "is this user / the user of this session inside the caller's
// scope" questions. found is false when the target row does not exist, so the
// caller can let the endpoint produce its normal 404.
type Store interface {
	UserInScope(ctx context.Context, s Scope, userID string) (inScope, found bool, err error)
	SessionUserInScope(ctx context.Context, s Scope, sessionID string) (inScope, found bool, err error)
}

type pgStore struct{ db *sqlx.DB }

// NewStore returns a Store backed by PostgreSQL.
func NewStore(db *sqlx.DB) Store { return &pgStore{db: db} }

func (p *pgStore) UserInScope(ctx context.Context, s Scope, userID string) (bool, bool, error) {
	q := withScope(`SELECT CASE WHEN @SCOPE@ THEN 1 ELSE 0 END AS in_scope FROM users u WHERE u.id = $1`, "u.id")
	return p.scan(ctx, "UserInScope", q, userID, s.Arg())
}

func (p *pgStore) SessionUserInScope(ctx context.Context, s Scope, sessionID string) (bool, bool, error) {
	q := withScope(`SELECT CASE WHEN @SCOPE@ THEN 1 ELSE 0 END AS in_scope FROM exam_sessions es WHERE es.id = $1`, "es.user_id")
	return p.scan(ctx, "SessionUserInScope", q, sessionID, s.Arg())
}

// withScope substitutes @SCOPE@ with the subtree predicate for userCol bound to $2.
func withScope(q, userCol string) string {
	return strings.ReplaceAll(q, "@SCOPE@", Predicate(userCol, "$2"))
}

func (p *pgStore) scan(ctx context.Context, op, q, id string, arg any) (bool, bool, error) {
	var n int
	if err := p.db.QueryRowContext(ctx, q, id, arg).Scan(&n); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, false, nil
		}
		return false, false, fmt.Errorf("deptscope: %s: %w", op, err)
	}
	return n == 1, true, nil
}
