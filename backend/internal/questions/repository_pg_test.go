package questions

// #593: real-Postgres tests of the question repository (CRUD transactions, versioning, filtered list, detail) and of
// the import/export repository (category path, similar stems, batch import, streamed export). They run the production
// migrations into a throwaway schema and are skipped unless TEST_DATABASE_URL is set (the backend CI job sets it).

import (
	"context"
	"errors"
	"testing"

	"github.com/bilimbaga/bilimbaga/internal/testpg"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type pgQuestionFixture struct {
	db      *sqlx.DB
	repo    Repository
	userID  string
	catID   string // root category "Pg Root"
	childID string // child category "Child" under the root
	tagA    string
	tagB    string
}

func newPGQuestionFixture(t *testing.T) pgQuestionFixture {
	t.Helper()
	base := testpg.Base(t)
	ctx := context.Background()
	db := testpg.NewSchema(t, base, "bb_593")
	applyUpMigrations(t, db, "../../migrations")

	var roleID string
	require.NoError(t, db.GetContext(ctx, &roleID, `SELECT id::text FROM roles ORDER BY name LIMIT 1`))
	fx := pgQuestionFixture{db: db, repo: NewRepository(db)}
	require.NoError(t, db.GetContext(ctx, &fx.userID,
		`INSERT INTO users (email, password_hash, full_name, role_id) VALUES ('q593@test.local', 'x', 'Q', $1) RETURNING id::text`, roleID))
	require.NoError(t, db.GetContext(ctx, &fx.catID, `INSERT INTO categories (name) VALUES ('Pg Root') RETURNING id::text`))
	require.NoError(t, db.GetContext(ctx, &fx.childID,
		`INSERT INTO categories (name, parent_id) VALUES ('Child', $1) RETURNING id::text`, fx.catID))
	require.NoError(t, db.GetContext(ctx, &fx.tagA, `INSERT INTO tags (name) VALUES ('Alpha') RETURNING id::text`))
	require.NoError(t, db.GetContext(ctx, &fx.tagB, `INSERT INTO tags (name) VALUES ('Beta') RETURNING id::text`))
	return fx
}

func (fx pgQuestionFixture) fullInput(stem string, tags ...string) CreateQuestionFullInput {
	weight := 2.0
	return CreateQuestionFullInput{
		CategoryID:    fx.childID,
		Difficulty:    "easy",
		Type:          "single",
		DefaultLocale: "en",
		CreatedBy:     fx.userID,
		Translations: map[string]TranslationInput{
			"en": {Stem: stem},
			"ru": {Stem: stem + " (ru)"},
		},
		AnswerOptions: []AnswerOptionInput{
			{SortOrder: 1, IsCorrect: true, LikertWeight: &weight,
				Translations: map[string]AnswerTranslationInput{"en": {Text: "Yes"}, "ru": {Text: "Da"}}},
			{SortOrder: 2, Translations: map[string]AnswerTranslationInput{"en": {Text: "No"}}},
		},
		TagIDs: tags,
	}
}

func TestRepository_RealPostgres_CreateFullThenGetWithDetails(t *testing.T) {
	fx := newPGQuestionFixture(t)
	ctx := context.Background()

	created, err := fx.repo.CreateFull(ctx, fx.fullInput("Capital of France?", fx.tagA))
	require.NoError(t, err)
	assert.Equal(t, "draft", created.Status)
	assert.Equal(t, 1, created.Version)
	assert.Equal(t, []string{"en", "ru"}, created.LocaleCoverage)
	assert.Len(t, created.AnswerOptions, 2)

	got, err := fx.repo.GetWithDetails(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, "Capital of France?", got.Translations["en"].Stem)
	assert.Equal(t, []string{fx.tagA}, got.TagIDs)
	require.Len(t, got.AnswerOptions, 2)
	assert.Equal(t, "Yes", got.AnswerOptions[0].Translations["en"].Text)
	assert.True(t, got.AnswerOptions[0].IsCorrect)

	_, err = fx.repo.GetWithDetails(ctx, "00000000-0000-4000-8000-00000000dead")
	assert.True(t, errors.Is(err, ErrQuestionNotFound), "got %v", err)
}

func TestRepository_RealPostgres_CreateFullRollsBackOnBadSubObject(t *testing.T) {
	fx := newPGQuestionFixture(t)
	ctx := context.Background()
	in := fx.fullInput("Rolled back?", "00000000-0000-4000-8000-00000000beef") // unknown tag -> FK violation

	_, err := fx.repo.CreateFull(ctx, in)
	require.Error(t, err)

	var n int
	require.NoError(t, fx.db.GetContext(ctx, &n, `SELECT count(*) FROM questions`))
	assert.Zero(t, n, "the question insert is rolled back with the failed tag insert")
}

func TestRepository_RealPostgres_UpdateInPlaceReplacesSubObjects(t *testing.T) {
	fx := newPGQuestionFixture(t)
	ctx := context.Background()
	created, err := fx.repo.CreateFull(ctx, fx.fullInput("Old stem?", fx.tagA))
	require.NoError(t, err)

	updated, err := fx.repo.UpdateInPlace(ctx, created.ID, UpdateQuestionInput{
		CategoryID:   fx.catID,
		Difficulty:   "hard",
		Translations: map[string]TranslationInput{"en": {Stem: "New stem?"}},
		AnswerOptions: []AnswerOptionInput{
			{SortOrder: 1, IsCorrect: true, Translations: map[string]AnswerTranslationInput{"en": {Text: "Only"}}},
		},
		TagIDs: []string{fx.tagB},
	})
	require.NoError(t, err)
	assert.Equal(t, "hard", updated.Difficulty)
	assert.Equal(t, fx.catID, updated.CategoryID)

	got, err := fx.repo.GetWithDetails(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{"en"}, got.LocaleCoverage, "the ru translation is gone")
	assert.Equal(t, "New stem?", got.Translations["en"].Stem)
	assert.Len(t, got.AnswerOptions, 1)
	assert.Equal(t, []string{fx.tagB}, got.TagIDs)

	_, err = fx.repo.UpdateInPlace(ctx, "00000000-0000-4000-8000-00000000dead", UpdateQuestionInput{CategoryID: fx.catID, Difficulty: "easy"})
	assert.True(t, errors.Is(err, ErrQuestionNotFound), "got %v", err)
}

func TestRepository_RealPostgres_CreateVersionFullArchivesPreviousAndChains(t *testing.T) {
	fx := newPGQuestionFixture(t)
	ctx := context.Background()
	v1, err := fx.repo.CreateFull(ctx, fx.fullInput("Versioned?"))
	require.NoError(t, err)

	v2, err := fx.repo.CreateVersionFull(ctx, v1.ID, UpdateQuestionInput{
		CategoryID:   fx.childID,
		Difficulty:   "medium",
		UpdatedBy:    fx.userID,
		Translations: map[string]TranslationInput{"en": {Stem: "Versioned v2?"}},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, v2.Version)
	require.NotNil(t, v2.ParentID)
	assert.Equal(t, v1.ID, *v2.ParentID)
	assert.Equal(t, "draft", v2.Status)

	old, err := fx.repo.GetByID(ctx, v1.ID)
	require.NoError(t, err)
	assert.Equal(t, "archived", old.Status)

	chain, err := fx.repo.GetVersionChain(ctx, v2.ID)
	require.NoError(t, err)
	assert.Len(t, chain, 2)

	_, err = fx.repo.CreateVersionFull(ctx, "00000000-0000-4000-8000-00000000dead", UpdateQuestionInput{CategoryID: fx.childID})
	assert.True(t, errors.Is(err, ErrQuestionNotFound), "got %v", err)
}

func TestRepository_RealPostgres_ListFilteredFiltersSortsAndHidesSuperseded(t *testing.T) {
	fx := newPGQuestionFixture(t)
	ctx := context.Background()

	a, err := fx.repo.CreateFull(ctx, fx.fullInput("Alpha question about networks?", fx.tagA))
	require.NoError(t, err)
	bIn := fx.fullInput("Beta question about storage?", fx.tagB)
	bIn.Difficulty = "hard"
	bIn.Translations = map[string]TranslationInput{"en": {Stem: "Beta question about storage?"}}
	b, err := fx.repo.CreateFull(ctx, bIn)
	require.NoError(t, err)
	_, err = fx.repo.CreateVersionFull(ctx, a.ID, UpdateQuestionInput{
		CategoryID: fx.childID, Difficulty: "easy", UpdatedBy: fx.userID,
		Translations: map[string]TranslationInput{"en": {Stem: "Alpha question v2?"}},
	})
	require.NoError(t, err)

	ids := func(items []*QuestionListItem) []string {
		out := make([]string, len(items))
		for i, it := range items {
			out[i] = it.ID
		}
		return out
	}
	list := func(f QuestionFilter) ([]*QuestionListItem, int) {
		t.Helper()
		if f.Page == 0 {
			f.Page = 1
		}
		if f.PerPage == 0 {
			f.PerPage = 50
		}
		items, total, err := fx.repo.ListFiltered(ctx, f)
		require.NoError(t, err)
		return items, total
	}

	items, total := list(QuestionFilter{})
	assert.Equal(t, 2, total, "the superseded v1 of Alpha is hidden")
	assert.NotContains(t, ids(items), a.ID)

	_, total = list(QuestionFilter{IncludeSuperseded: true})
	assert.Equal(t, 3, total)

	items, total = list(QuestionFilter{Difficulties: []string{"hard"}})
	assert.Equal(t, 1, total)
	assert.Equal(t, []string{b.ID}, ids(items))
	assert.Equal(t, "Beta question about storage?", items[0].StemPreview)

	items, _ = list(QuestionFilter{TagIDs: []string{fx.tagB}})
	assert.Equal(t, []string{b.ID}, ids(items))

	locale := "ru"
	_, total = list(QuestionFilter{Locale: &locale})
	assert.Zero(t, total, "the only ru translation belongs to the superseded v1")
	_, total = list(QuestionFilter{LocaleMissing: &locale, IncludeSuperseded: true})
	assert.Equal(t, 2, total)

	typ := "single"
	_, total = list(QuestionFilter{Type: &typ, CategoryID: &fx.childID, Statuses: []string{"draft"}})
	assert.Equal(t, 2, total)

	items, total = list(QuestionFilter{Sort: "difficulty", Order: "asc", PerPage: 1, Page: 2})
	assert.Equal(t, 2, total)
	assert.Len(t, items, 1)

	items, total = list(QuestionFilter{Search: "storage"})
	assert.Equal(t, 1, total)
	assert.Equal(t, []string{b.ID}, ids(items))
}

func TestImportExportRepository_RealPostgres_CategoryPathAndTags(t *testing.T) {
	fx := newPGQuestionFixture(t)
	ctx := context.Background()
	repo := fx.repo.(*postgresRepository)

	id, err := repo.ResolveCategoryPath(ctx, " pg root / CHILD ")
	require.NoError(t, err)
	assert.Equal(t, fx.childID, id)

	_, err = repo.ResolveCategoryPath(ctx, "Pg Root/Missing")
	require.Error(t, err)
	_, err = repo.ResolveCategoryPath(ctx, "Pg Root//Child")
	require.Error(t, err)

	tagID, err := repo.TagNameToID(ctx, "ALPHA")
	require.NoError(t, err)
	assert.Equal(t, fx.tagA, tagID)
	_, err = repo.TagNameToID(ctx, "nope")
	require.Error(t, err)

	m, err := repo.TagIDByName(ctx, []string{"alpha", "BETA", "ghost"})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"alpha": fx.tagA, "beta": fx.tagB}, m)
	m, err = repo.TagIDByName(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, m)
}

func TestImportExportRepository_RealPostgres_ImportBatchThenStreamExport(t *testing.T) {
	fx := newPGQuestionFixture(t)
	ctx := context.Background()
	repo := fx.repo.(*postgresRepository)

	rows := []ImportRow{
		{RowNumber: 1, Type: "single", Difficulty: "easy", CategoryPath: fx.childID, DefaultLocale: "en",
			Translations: map[string]TranslationInput{"en": {Stem: "Imported one?"}},
			AnswerOptions: []AnswerOptionInput{
				{SortOrder: 1, IsCorrect: true, Translations: map[string]AnswerTranslationInput{"en": {Text: "A"}}},
			},
			Tags: []string{fx.tagA}},
		{RowNumber: 2, Type: "truefalse", Difficulty: "hard", CategoryPath: fx.catID, DefaultLocale: "en",
			Translations: map[string]TranslationInput{"en": {Stem: "Imported two?"}}},
	}
	ids, err := repo.ImportBatch(ctx, rows, fx.userID)
	require.NoError(t, err)
	require.Len(t, ids, 2)

	// A bad row rolls the whole batch back.
	_, err = repo.ImportBatch(ctx, []ImportRow{
		{RowNumber: 1, Type: "single", Difficulty: "easy", CategoryPath: fx.catID, DefaultLocale: "en",
			Translations: map[string]TranslationInput{"en": {Stem: "Never lands?"}}},
		{RowNumber: 2, Type: "bogus-type", Difficulty: "easy", CategoryPath: fx.catID, DefaultLocale: "en"},
	}, fx.userID)
	require.Error(t, err)
	var n int
	require.NoError(t, fx.db.GetContext(ctx, &n, `SELECT count(*) FROM questions`))
	assert.Equal(t, 2, n)

	collect := func(f ExportFilter) []*ExportRow {
		t.Helper()
		var out []*ExportRow
		require.NoError(t, repo.StreamExport(ctx, f, func(r *ExportRow) error {
			out = append(out, r)
			return nil
		}))
		return out
	}

	all := collect(ExportFilter{})
	require.Len(t, all, 2)
	assert.Equal(t, "Pg Root/Child", all[0].CategoryPath)
	assert.Equal(t, "Imported one?", all[0].Translations["en"].Stem)
	assert.Len(t, all[0].AnswerOptions, 1)
	assert.Equal(t, []string{"Alpha"}, all[0].Tags)

	hard := collect(ExportFilter{Difficulties: []string{"hard"}})
	require.Len(t, hard, 1)
	assert.Equal(t, "truefalse", hard[0].Type)

	typ, loc := "single", "en"
	assert.Len(t, collect(ExportFilter{Type: &typ, Locale: &loc, CategoryID: &fx.childID, Statuses: []string{"draft"}, TagIDs: []string{fx.tagA}}), 1)
	assert.Len(t, collect(ExportFilter{IDs: []string{ids[1]}}), 1)

	stop := errors.New("stop")
	err = repo.StreamExport(ctx, ExportFilter{}, func(*ExportRow) error { return stop })
	assert.True(t, errors.Is(err, stop), "the callback error is returned, got %v", err)
}

func TestImportExportRepository_RealPostgres_FindSimilarStemsOnlyMatchesActive(t *testing.T) {
	fx := newPGQuestionFixture(t)
	ctx := context.Background()
	repo := fx.repo.(*postgresRepository)

	created, err := fx.repo.CreateFull(ctx, fx.fullInput("What is the capital of France?"))
	require.NoError(t, err)

	none, err := repo.FindSimilarStems(ctx, nil, "en")
	require.NoError(t, err)
	assert.Empty(t, none)

	res, err := repo.FindSimilarStems(ctx, []string{"What is the capital of France?"}, "en")
	require.NoError(t, err)
	assert.Empty(t, res, "a draft question is not a duplicate candidate")

	_, err = fx.db.ExecContext(ctx, `UPDATE questions SET status = 'active' WHERE id = $1`, created.ID)
	require.NoError(t, err)
	res, err = repo.FindSimilarStems(ctx, []string{"What is the capital of France?", "Unrelated xyzzy"}, "en")
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, created.ID, res["What is the capital of France?"].QuestionID)
	assert.InDelta(t, 1.0, res["What is the capital of France?"].Score, 0.001)
}
