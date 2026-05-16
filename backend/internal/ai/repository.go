package ai

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// Repository defines the data-access operations required by the AI service.
type Repository interface {
	// CountAIUsageLastHour returns the number of AI calls made by userID for feature
	// in the last 60 minutes.
	CountAIUsageLastHour(ctx context.Context, userID, feature string) (int, error)

	// LogUsage inserts a row into ai_usage_log.
	LogUsage(ctx context.Context, log UsageLog) error

	// GetCategoryName returns the name of a category by its UUID.
	GetCategoryName(ctx context.Context, categoryID string) (string, error)
}

type postgresRepository struct {
	db *sqlx.DB
}

// NewRepository returns a Repository backed by PostgreSQL.
func NewRepository(db *sqlx.DB) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) CountAIUsageLastHour(ctx context.Context, userID, feature string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ai_usage_log
		 WHERE user_id = $1 AND feature = $2 AND created_at >= $3`,
		userID, feature, time.Now().UTC().Add(-time.Hour),
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("ai: CountAIUsageLastHour: %w", err)
	}
	return count, nil
}

func (r *postgresRepository) LogUsage(ctx context.Context, log UsageLog) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO ai_usage_log (user_id, feature, tokens_used, model, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		log.UserID, log.Feature, log.TokensUsed, log.Model, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("ai: LogUsage: %w", err)
	}
	return nil
}

func (r *postgresRepository) GetCategoryName(ctx context.Context, categoryID string) (string, error) {
	var name string
	err := r.db.QueryRowContext(ctx,
		`SELECT name FROM categories WHERE id = $1`, categoryID,
	).Scan(&name)
	if err != nil {
		return "", fmt.Errorf("ai: GetCategoryName: %w", err)
	}
	return name, nil
}
