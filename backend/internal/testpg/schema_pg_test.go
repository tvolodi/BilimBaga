package testpg

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// #547: two test packages share one database. A schema's gin_trgm_ops index must survive another schema's drop, and
// pg_trgm must live in public, where NewSchema installs it. Skipped unless TEST_DATABASE_URL is set.
func TestNewSchema_RealPostgres_TrgmSurvivesAnotherSchemasDrop(t *testing.T) {
	base := Base(t)
	ctx := context.Background()
	first := NewSchema(t, base, "bb_547a")
	second := NewSchema(t, base, "bb_547b")

	const trgmIndex = `
		CREATE TABLE question_translations (stem text);
		CREATE INDEX question_translations_stem_trgm ON question_translations USING GIN (stem gin_trgm_ops);`

	// The first schema builds its index, then the second builds the same index while the first still exists.
	_, err := first.ExecContext(ctx, trgmIndex)
	require.NoError(t, err)
	_, err = second.ExecContext(ctx, trgmIndex)
	require.NoError(t, err)

	var firstSchema, extSchema string
	require.NoError(t, first.GetContext(ctx, &firstSchema, `SELECT current_schema()`))
	require.NoError(t, second.GetContext(ctx, &extSchema,
		`SELECT extnamespace::regnamespace::text FROM pg_extension WHERE extname = 'pg_trgm'`))
	require.Equal(t, "public", extSchema, "pg_trgm must live in public, so no test schema owns it")

	// The first schema's cleanup drops it with CASCADE. pg_trgm and the second schema's index must survive.
	_, err = second.ExecContext(ctx, `DROP SCHEMA `+firstSchema+` CASCADE`)
	require.NoError(t, err)
	var sim float64
	require.NoError(t, second.GetContext(ctx, &sim, `SELECT similarity('abc', 'abd')`))
	require.Greater(t, sim, 0.0)
	_, err = second.ExecContext(ctx, `
		CREATE TABLE after_drop (stem text);
		CREATE INDEX after_drop_stem_trgm ON after_drop USING GIN (stem gin_trgm_ops);`)
	require.NoError(t, err)
}
