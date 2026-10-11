package questions

// #514: real-Postgres test of the translation repository's SQL. It runs the real migrations into a throwaway schema,
// so the constraints the ON CONFLICT clauses rely on are the production ones. Skipped unless TEST_DATABASE_URL is
// set; the backend CI job sets it. Run locally with:
//
//	cd backend && TEST_DATABASE_URL='postgres://USER:PASS@HOST:PORT/DB?sslmode=disable' \
//	    go test ./internal/questions -run TranslationRepository_RealPostgres -v

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/bilimbaga/bilimbaga/internal/testpg"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type pgTranslationFixture struct {
	db         *sqlx.DB
	repo       TranslationRepository
	questionID string
	first      string // answer option with sort_order 1
	second     string // answer option with sort_order 2
}

// newPGTranslationFixture builds a throwaway schema with the production migrations, seeds one question with two
// options, and drops the schema when the test ends.
func newPGTranslationFixture(t *testing.T) pgTranslationFixture {
	t.Helper()
	base := testpg.Base(t)
	migrations, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	require.NoError(t, err)

	ctx := context.Background()
	db := testpg.NewSchema(t, base, "bb_514")
	applyUpMigrations(t, db, migrations)

	var roleID, catID, userID, qID string
	require.NoError(t, db.GetContext(ctx, &roleID, `SELECT id::text FROM roles ORDER BY name LIMIT 1`))
	require.NoError(t, db.GetContext(ctx, &catID, `INSERT INTO categories (name) VALUES ('tr-test') RETURNING id::text`))
	require.NoError(t, db.GetContext(ctx, &userID,
		`INSERT INTO users (email, password_hash, full_name, role_id) VALUES ('tr@test.local', 'x', 'T', $1) RETURNING id::text`,
		roleID))
	require.NoError(t, db.GetContext(ctx, &qID,
		`INSERT INTO questions (category_id, difficulty, type, default_locale, created_by)
		 VALUES ($1, 'easy', 'single', 'en', $2) RETURNING id::text`, catID, userID))

	// The second option is inserted first, so the order must come from sort_order and not from insertion order.
	var second, first string
	require.NoError(t, db.GetContext(ctx, &second,
		`INSERT INTO answer_options (question_id, sort_order) VALUES ($1, 2) RETURNING id::text`, qID))
	require.NoError(t, db.GetContext(ctx, &first,
		`INSERT INTO answer_options (question_id, sort_order) VALUES ($1, 1) RETURNING id::text`, qID))

	return pgTranslationFixture{db: db, repo: NewTranslationRepository(db), questionID: qID, first: first, second: second}
}

// applyUpMigrations runs every *.up.sql file in dir, in filename order, the order the migrator uses. It reads the
// files directly because the migrator builds a file:// URL from the path, which a Windows drive path breaks.
func applyUpMigrations(t *testing.T, db *sqlx.DB, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	var names []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".up.sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	require.NotEmpty(t, names, "no up migrations found in %s", dir)
	for _, name := range names {
		sqlText, err := os.ReadFile(filepath.Join(dir, name))
		require.NoError(t, err)
		_, err = db.Exec(string(sqlText))
		require.NoError(t, err, "migration %s", name)
	}
}

func TestTranslationRepository_RealPostgres_OptionIDsFollowSortOrder(t *testing.T) {
	fx := newPGTranslationFixture(t)
	ids, err := fx.repo.ListAnswerOptionIDs(context.Background(), fx.questionID)
	require.NoError(t, err)
	assert.Equal(t, []string{fx.first, fx.second}, ids)
}

func TestTranslationRepository_RealPostgres_GetQuestionFoundAndMissing(t *testing.T) {
	fx := newPGTranslationFixture(t)
	ctx := context.Background()

	q, err := fx.repo.GetQuestion(ctx, fx.questionID)
	require.NoError(t, err)
	assert.Equal(t, fx.questionID, q.ID)
	assert.Equal(t, "en", q.DefaultLocale)

	_, err = fx.repo.GetQuestion(ctx, "00000000-0000-4000-8000-00000000dead")
	assert.True(t, errors.Is(err, ErrQuestionNotFound), "an unknown question is ErrQuestionNotFound, got %v", err)
}

func TestTranslationRepository_RealPostgres_UpsertReplacesStemAndKeepsOtherOptionTexts(t *testing.T) {
	fx := newPGTranslationFixture(t)
	ctx := context.Background()
	expl := "Because"

	written, err := fx.repo.UpsertLocale(ctx, fx.questionID, "en", UpsertTranslationInput{
		Stem:        "Q1?",
		Explanation: &expl,
		Options: []AnswerTextTranslation{
			{OptionID: fx.first, Text: "A"},
			{OptionID: fx.second, Text: "B"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "Q1?", written.Stem)

	got, err := fx.repo.LoadLocaleTranslation(ctx, fx.questionID, "en")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "Q1?", got.Stem)
	if assert.NotNil(t, got.Explanation) {
		assert.Equal(t, "Because", *got.Explanation)
	}
	assert.False(t, got.UpdatedAt.IsZero(), "the database sets updated_at")
	assert.Equal(t, []AnswerTextTranslation{{OptionID: fx.first, Text: "A"}, {OptionID: fx.second, Text: "B"}}, got.Options,
		"options come back in sort_order")

	// A second upsert replaces the stem in place (one row per locale) and updates only the options it is given.
	_, err = fx.repo.UpsertLocale(ctx, fx.questionID, "en", UpsertTranslationInput{
		Stem:    "Q2?",
		Options: []AnswerTextTranslation{{OptionID: fx.second, Text: "B2"}},
	})
	require.NoError(t, err)

	all, err := fx.repo.LoadAllTranslations(ctx, fx.questionID)
	require.NoError(t, err)
	require.Len(t, all, 1, "one translation row per locale")
	assert.Equal(t, "Q2?", all["en"].Stem)
	assert.Nil(t, all["en"].Explanation, "an upsert without an explanation clears it")
	assert.Equal(t, []AnswerTextTranslation{{OptionID: fx.first, Text: "A"}, {OptionID: fx.second, Text: "B2"}}, all["en"].Options,
		"an option that is not in the input keeps its text; the service requires every option")
}

func TestTranslationRepository_RealPostgres_LoadLocaleAbsentIsNil(t *testing.T) {
	fx := newPGTranslationFixture(t)
	got, err := fx.repo.LoadLocaleTranslation(context.Background(), fx.questionID, "ru")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestTranslationRepository_RealPostgres_LoadAllSurfacesOrphanOptionTextPerLocale(t *testing.T) {
	fx := newPGTranslationFixture(t)
	ctx := context.Background()
	_, err := fx.repo.UpsertLocale(ctx, fx.questionID, "en", UpsertTranslationInput{
		Stem:    "Q?",
		Options: []AnswerTextTranslation{{OptionID: fx.first, Text: "A"}},
	})
	require.NoError(t, err)
	// Option text for kk with no question_translations row for kk: an orphan that locale coverage must still report.
	_, err = fx.db.ExecContext(ctx,
		`INSERT INTO answer_translations (option_id, locale, text) VALUES ($1, 'kk', 'Ә')`, fx.first)
	require.NoError(t, err)

	all, err := fx.repo.LoadAllTranslations(ctx, fx.questionID)
	require.NoError(t, err)
	require.Contains(t, all, "en")
	require.Contains(t, all, "kk")
	assert.Equal(t, "Q?", all["en"].Stem)
	assert.Empty(t, all["kk"].Stem, "no stem exists for the orphan locale")
	assert.Equal(t, []AnswerTextTranslation{{OptionID: fx.first, Text: "Ә"}}, all["kk"].Options)
}

func TestTranslationRepository_RealPostgres_DeleteLocaleRemovesOnlyThatLocale(t *testing.T) {
	fx := newPGTranslationFixture(t)
	ctx := context.Background()
	for _, locale := range []string{"en", "ru"} {
		_, err := fx.repo.UpsertLocale(ctx, fx.questionID, locale, UpsertTranslationInput{
			Stem:    "stem-" + locale,
			Options: []AnswerTextTranslation{{OptionID: fx.first, Text: "text-" + locale}},
		})
		require.NoError(t, err)
	}

	require.NoError(t, fx.repo.DeleteLocale(ctx, fx.questionID, "en"))

	gone, err := fx.repo.LoadLocaleTranslation(ctx, fx.questionID, "en")
	require.NoError(t, err)
	assert.Nil(t, gone)

	var leftover int
	require.NoError(t, fx.db.GetContext(ctx, &leftover,
		`SELECT count(*) FROM answer_translations WHERE locale = 'en'`))
	assert.Zero(t, leftover, "the en option texts go with the en locale")

	kept, err := fx.repo.LoadLocaleTranslation(ctx, fx.questionID, "ru")
	require.NoError(t, err)
	if assert.NotNil(t, kept) {
		assert.Equal(t, "stem-ru", kept.Stem)
		assert.Equal(t, []AnswerTextTranslation{{OptionID: fx.first, Text: "text-ru"}}, kept.Options)
	}
}

func TestTranslationRepository_RealPostgres_DeleteMissingLocaleIsNoError(t *testing.T) {
	fx := newPGTranslationFixture(t)
	assert.NoError(t, fx.repo.DeleteLocale(context.Background(), fx.questionID, "zz"))
}
