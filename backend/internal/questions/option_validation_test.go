package questions

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ISS-173: answer option text validation (service, handler, import).

func optInput(order int, texts map[string]string) AnswerOptionInput {
	tr := make(map[string]AnswerTranslationInput, len(texts))
	for l, txt := range texts {
		tr[l] = AnswerTranslationInput{Text: txt}
	}
	return AnswerOptionInput{SortOrder: order, Translations: tr}
}

func TestOptionTextFieldErrors(t *testing.T) {
	cases := []struct {
		name    string
		qType   string
		locale  string
		opts    []AnswerOptionInput
		wantIdx []int
	}{
		{"valid single", "single", "en", []AnswerOptionInput{optInput(1, map[string]string{"en": "A"}), optInput(2, map[string]string{"en": "B"})}, nil},
		{"empty default text", "single", "en", []AnswerOptionInput{optInput(1, map[string]string{"en": "A"}), optInput(2, map[string]string{"en": ""})}, []int{1}},
		{"whitespace only", "multiple", "en", []AnswerOptionInput{optInput(1, map[string]string{"en": "  \t "}), optInput(2, map[string]string{"en": "B"})}, []int{0}},
		{"missing default locale entry", "single", "en", []AnswerOptionInput{optInput(1, map[string]string{"kk": "A"}), optInput(2, map[string]string{"en": "B"})}, []int{0}},
		{"nil translations", "single", "en", []AnswerOptionInput{{SortOrder: 1}, optInput(2, map[string]string{"en": "B"})}, []int{0}},
		{"non-default locale empty ok", "single", "en", []AnswerOptionInput{optInput(1, map[string]string{"en": "A", "kk": ""}), optInput(2, map[string]string{"en": "B", "ru": " "})}, nil},
		{"truefalse valid", "truefalse", "en", []AnswerOptionInput{optInput(1, map[string]string{"en": "True"}), optInput(2, map[string]string{"en": "False"})}, nil},
		{"truefalse empty", "truefalse", "en", []AnswerOptionInput{optInput(1, map[string]string{"en": "True"}), optInput(2, map[string]string{"en": ""})}, []int{1}},
		{"likert valid", "likert", "en", []AnswerOptionInput{optInput(1, map[string]string{"en": "Agree"}), optInput(2, map[string]string{"en": "Disagree"})}, nil},
		{"likert empty", "likert", "en", []AnswerOptionInput{optInput(1, map[string]string{"en": ""}), optInput(2, map[string]string{"en": "Disagree"})}, []int{0}},
		{"shorttext no options", "shorttext", "en", nil, nil},
		{"shorttext ignored", "shorttext", "en", []AnswerOptionInput{optInput(1, map[string]string{"en": ""})}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			errs := optionTextFieldErrors(tc.qType, tc.locale, tc.opts)
			require.Len(t, errs, len(tc.wantIdx))
			for i, idx := range tc.wantIdx {
				assert.Equal(t, fmt.Sprintf("answer_options[%d].translations.%s.text", idx, tc.locale), errs[i].Field)
			}
		})
	}
}

func createInput(opts ...AnswerOptionInput) CreateQuestionFullInput {
	return CreateQuestionFullInput{
		CategoryID: "cat-1", Difficulty: "easy", Type: "single", DefaultLocale: "en", CreatedBy: "u",
		Translations:  map[string]TranslationInput{"en": {Stem: "s"}},
		AnswerOptions: opts,
	}
}

func TestCreateQuestionFull_EmptyDefaultLocaleOptionText_Rejected(t *testing.T) {
	svc := NewService(newMockRepo())
	_, err := svc.CreateQuestionFull(context.Background(), createInput(
		optInput(1, map[string]string{"en": "A"}), optInput(2, map[string]string{"en": " "})))
	require.ErrorIs(t, err, ErrInvalidOptionText)
	var oe *OptionValidationError
	require.True(t, errors.As(err, &oe))
	require.Len(t, oe.Fields, 1)
	assert.Contains(t, oe.Fields[0].Field, "answer_options[1]")
}

func TestCreateQuestionFull_ValidOptions_Passes(t *testing.T) {
	svc := NewService(newMockRepo())
	_, err := svc.CreateQuestionFull(context.Background(), createInput(
		optInput(1, map[string]string{"en": "A", "kk": ""}), optInput(2, map[string]string{"en": "B"})))
	require.NoError(t, err)
}

func TestUpdateQuestion_EmptyDefaultLocaleOptionText_Rejected(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)
	seedSingleQuestion(repo, "q-single") // default locale "en"
	_, err := svc.UpdateQuestion(context.Background(), "q-single", UpdateQuestionInput{
		CategoryID: "cat-1", Difficulty: "easy", UpdatedBy: "u",
		AnswerOptions: []AnswerOptionInput{optInput(1, map[string]string{"en": ""}), optInput(2, map[string]string{"en": "B"})},
	})
	require.ErrorIs(t, err, ErrInvalidOptionText)
}

func TestUpdateQuestion_OtherLocaleEmptyOptionText_NotRejected(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)
	seedSingleQuestion(repo, "q-single")
	_, err := svc.UpdateQuestion(context.Background(), "q-single", UpdateQuestionInput{
		CategoryID: "cat-1", Difficulty: "easy", UpdatedBy: "u",
		AnswerOptions: []AnswerOptionInput{optInput(1, map[string]string{"en": "A", "kk": ""}), optInput(2, map[string]string{"en": "B"})},
	})
	assert.False(t, errors.Is(err, ErrInvalidOptionText), "unexpected option validation error: %v", err)
}

func TestValidateCreateRequest_EmptyOptionText(t *testing.T) {
	mk := func(typ, text string) createQuestionReq {
		w := 1.0
		return createQuestionReq{
			CategoryID: "c", Difficulty: "easy", Type: typ, DefaultLocale: "en",
			Translations: map[string]translationReq{"en": {Stem: "s"}},
			AnswerOptions: []answerOptionReq{
				{SortOrder: 1, LikertWeight: &w, Translations: map[string]answerTranslationReq{"en": {Text: "A"}}},
				{SortOrder: 2, LikertWeight: &w, Translations: map[string]answerTranslationReq{"en": {Text: text}}},
			},
		}
	}
	for _, typ := range []string{"single", "multiple", "truefalse", "likert"} {
		errs := validateCreateRequest(mk(typ, ""))
		if assert.Len(t, errs, 1, typ) {
			assert.True(t, strings.Contains(errs[0].Field, "answer_options[1]"))
		}
		assert.Len(t, validateCreateRequest(mk(typ, "   ")), 1, typ)
		assert.Empty(t, validateCreateRequest(mk(typ, "B")), typ)
	}
}

func TestImport_DryRun_RejectsEmptyOptionText(t *testing.T) {
	svc := NewService(newMockRepo())

	blank := buildValidRow(1)
	blank.AnswerOptions[1].Translations = map[string]AnswerTranslationInput{"kk": {Text: "  "}}
	noDefault := buildValidRow(2)
	noDefault.AnswerOptions[0].Translations = map[string]AnswerTranslationInput{"ru": {Text: "A"}}
	otherLocaleBlank := buildValidRow(3)
	otherLocaleBlank.AnswerOptions[0].Translations["ru"] = AnswerTranslationInput{Text: ""}

	report, _, err := svc.ValidateAndImport(context.Background(), []ImportRow{blank, noDefault, otherLocaleBlank}, true, "actor-1")
	require.NoError(t, err)
	assert.Equal(t, 1, report.ValidCount)
	require.Len(t, report.ErrorRows, 2)
	assert.Equal(t, 1, report.ErrorRows[0].Row)
	assert.Contains(t, report.ErrorRows[0].Errors[0], "answer_options[1]")
	assert.Equal(t, 2, report.ErrorRows[1].Row)
}

func postCreate(h *Handler, body string) *httptest.ResponseRecorder {
	req := withQAuthCtx(httptest.NewRequest(http.MethodPost, "/api/v1/questions", strings.NewReader(body)), "actor-1")
	w := httptest.NewRecorder()
	h.Create(w, req)
	return w
}

func TestQHandlerCreate_EmptyOptionText_Returns422(t *testing.T) {
	for name, text := range map[string]string{"empty": "", "whitespace": "   "} {
		t.Run(name, func(t *testing.T) {
			h := NewHandler(&mockQService{}, nil) // service must not be reached
			w := postCreate(h, `{"category_id":"cat-1","difficulty":"easy","type":"single","default_locale":"en",
				"translations":{"en":{"stem":"Q?"}},
				"answer_options":[
					{"sort_order":1,"is_correct":true,"translations":{"en":{"text":"A"}}},
					{"sort_order":2,"is_correct":false,"translations":{"en":{"text":"`+text+`"}}}]}`)
			assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
			assert.Contains(t, w.Body.String(), "ERR_VALIDATION")
			assert.Contains(t, w.Body.String(), "answer_options[1].translations.en.text")
		})
	}
}

func TestQHandlerCreate_BodyKeyInsteadOfText_Returns422(t *testing.T) {
	h := NewHandler(&mockQService{}, nil)
	w := postCreate(h, `{"category_id":"cat-1","difficulty":"easy","type":"truefalse","default_locale":"en",
		"translations":{"en":{"stem":"Q?"}},
		"answer_options":[
			{"sort_order":1,"is_correct":true,"translations":{"en":{"body":"True"}}},
			{"sort_order":2,"is_correct":false,"translations":{"en":{"body":"False"}}}]}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "answer_options[0].translations.en.text")
}

func TestQHandlerCreate_NonDefaultLocaleEmptyOption_Returns201(t *testing.T) {
	svc := &mockQService{
		createFullFn: func(_ context.Context, _ CreateQuestionFullInput) (*QuestionDetail, error) {
			return sampleDetail("q-1"), nil
		},
	}
	w := postCreate(NewHandler(svc, nil), `{"category_id":"cat-1","difficulty":"easy","type":"likert","default_locale":"en",
		"translations":{"en":{"stem":"Q?"}},
		"answer_options":[
			{"sort_order":1,"likert_weight":1,"translations":{"en":{"text":"Agree"},"kk":{"text":""}}},
			{"sort_order":2,"likert_weight":2,"translations":{"en":{"text":"Disagree"}}}]}`)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestQHandlerCreate_ServiceOptionError_Returns422(t *testing.T) {
	svc := &mockQService{
		createFullFn: func(_ context.Context, _ CreateQuestionFullInput) (*QuestionDetail, error) {
			return nil, fmt.Errorf("wrap: %w", &OptionValidationError{Fields: []fieldError{{Field: "answer_options[0].translations.en.text", Message: "bad"}}})
		},
	}
	w := postCreate(NewHandler(svc, nil), `{"category_id":"cat-1","difficulty":"easy","type":"single","default_locale":"en",
		"translations":{"en":{"stem":"Q?"}},
		"answer_options":[{"sort_order":1,"translations":{"en":{"text":"A"}}},{"sort_order":2,"translations":{"en":{"text":"B"}}}]}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestQHandlerUpdate_EmptyOptionText_Returns422(t *testing.T) {
	svc := &mockQService{
		updateFn: func(_ context.Context, _ string, _ UpdateQuestionInput) (*Question, error) {
			return nil, fmt.Errorf("questions: UpdateQuestion: %w", &OptionValidationError{Fields: []fieldError{{Field: "answer_options[1].translations.en.text", Message: "option 2 must have non-empty text"}}})
		},
	}
	h := NewHandler(svc, nil)
	body := `{"category_id":"cat-1","difficulty":"medium","translations":{},"answer_options":[{"sort_order":1,"translations":{"en":{"text":""}}}]}`
	req := withQChiParam(withQAuthCtx(httptest.NewRequest(http.MethodPut, "/api/v1/questions/q-1", strings.NewReader(body)), "actor-1"), "id", "q-1")
	w := httptest.NewRecorder()
	h.Update(w, req)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "ERR_VALIDATION")
	assert.Contains(t, w.Body.String(), "answer_options[1]")
}

// ── ISS-173b: translations PUT and activation ────────────────────────────────

func seedTypedQuestion(repo *mockTranslationRepo, id, qType, defLocale string, optionIDs ...string) {
	seedQuestion(repo, id, defLocale, optionIDs...)
	repo.questions[id].Type = qType
}

func TestTranslationUpsert_DefaultLocaleOptionText(t *testing.T) {
	tenant := &mockTenant{defaultLocale: "kk", availableLocales: []string{"kk", "en"}}
	cases := []struct {
		name    string
		qType   string
		locale  string
		texts   []string
		wantIdx []int // expected failing option indexes; nil = success
	}{
		{"blank default rejected", "single", "kk", []string{"A", ""}, []int{1}},
		{"whitespace default rejected", "multiple", "kk", []string{"  ", "B"}, []int{0}},
		{"truefalse blank rejected", "truefalse", "kk", []string{"", ""}, []int{0, 1}},
		{"likert blank rejected", "likert", "kk", []string{"x", ""}, []int{1}},
		{"blank non-default ok", "single", "en", []string{"", " "}, nil},
		{"valid default passes", "single", "kk", []string{"A", "B"}, nil},
		{"shorttext unaffected", "shorttext", "kk", []string{"", ""}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newMockTranslationRepo()
			seedTypedQuestion(repo, "q-1", tc.qType, "kk", "o1", "o2")
			svc := NewTranslationService(repo, tenant)
			_, _, err := svc.Upsert(context.Background(), "q-1", tc.locale, UpsertTranslationInput{
				Stem: "Stem",
				Options: []AnswerTextTranslation{
					{OptionID: "o1", Text: tc.texts[0]}, {OptionID: "o2", Text: tc.texts[1]},
				},
			})
			if tc.wantIdx == nil {
				require.NoError(t, err)
				assert.Equal(t, 1, repo.upsertCalls)
				return
			}
			require.Error(t, err)
			assert.True(t, errors.Is(err, ErrInvalidOptionText))
			var oe *OptionValidationError
			require.True(t, errors.As(err, &oe))
			var got []string
			for _, f := range oe.Fields {
				got = append(got, f.Field)
			}
			var want []string
			for _, i := range tc.wantIdx {
				want = append(want, fmt.Sprintf("answer_options[%d].translations.kk.text", i))
			}
			assert.Equal(t, want, got)
			assert.Equal(t, 0, repo.upsertCalls, "nothing persisted on rejection")
		})
	}
}

func TestTranslationHandlerUpsert_BlankDefaultOption_Returns422(t *testing.T) {
	svc := &mockTranslationService{
		upsertFn: func(_ context.Context, _, _ string, _ UpsertTranslationInput) (*LocaleTranslation, *LocaleTranslation, error) {
			return nil, nil, fmt.Errorf("translations.Upsert: %w", &OptionValidationError{Fields: []fieldError{
				{Field: "answer_options[1].translations.kk.text", Message: "blank"},
			}})
		},
	}
	h := NewTranslationHandler(svc, nil)
	body := `{"stem":"S","options":[{"option_id":"o1","text":"A"},{"option_id":"o2","text":""}]}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/questions/q-1/translations/kk", strings.NewReader(body))
	req = withIDLocaleParam(req, "q-1", "kk")
	w := httptest.NewRecorder()
	h.Upsert(w, req)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), `"ERR_VALIDATION"`)
	assert.Contains(t, w.Body.String(), "answer_options[1].translations.kk.text")
}

func activateSvc(qType string, status string, opts []AnswerOptionDetail) (Service, *mockRepository) {
	repo := newMockRepo()
	repo.questions["q-1"] = &Question{ID: "q-1", Type: qType, Status: status, DefaultLocale: "kk"}
	repo.detailOptions = map[string][]AnswerOptionDetail{"q-1": opts}
	return NewService(repo), repo
}

func optDetail(texts map[string]string) AnswerOptionDetail {
	tr := map[string]AnswerTranslationDetail{}
	for l, txt := range texts {
		tr[l] = AnswerTranslationDetail{Text: txt}
	}
	return AnswerOptionDetail{Translations: tr}
}

func TestTransitionStatus_ActivateBlankOptionText(t *testing.T) {
	cases := []struct {
		name    string
		qType   string
		opts    []AnswerOptionDetail
		wantIdx []int
	}{
		{"blank default rejected", "single", []AnswerOptionDetail{optDetail(map[string]string{"kk": "A"}), optDetail(map[string]string{"kk": " "})}, []int{1}},
		{"missing default entry rejected", "multiple", []AnswerOptionDetail{optDetail(map[string]string{"en": "A"}), optDetail(map[string]string{"kk": "B"})}, []int{0}},
		{"valid activates", "single", []AnswerOptionDetail{optDetail(map[string]string{"kk": "A"}), optDetail(map[string]string{"kk": "B", "en": ""})}, nil},
		{"shorttext unaffected", "shorttext", nil, nil},
		{"shorttext with stray blank unaffected", "shorttext", []AnswerOptionDetail{optDetail(map[string]string{"kk": ""})}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo := activateSvc(tc.qType, "review", tc.opts)
			q, err := svc.TransitionStatus(context.Background(), "q-1", "active")
			if tc.wantIdx == nil {
				require.NoError(t, err)
				assert.Equal(t, "active", q.Status)
				return
			}
			require.Error(t, err)
			assert.True(t, errors.Is(err, ErrInvalidOptionText))
			var oe *OptionValidationError
			require.True(t, errors.As(err, &oe))
			require.Len(t, oe.Fields, len(tc.wantIdx))
			for i, idx := range tc.wantIdx {
				assert.Equal(t, fmt.Sprintf("answer_options[%d].translations.kk.text", idx), oe.Fields[i].Field)
			}
			assert.Equal(t, "review", repo.questions["q-1"].Status, "status must not change on rejection")
		})
	}
}

func TestTransitionStatus_NonActivateTargetsIgnoreBlankOptions(t *testing.T) {
	blank := []AnswerOptionDetail{optDetail(map[string]string{"kk": ""})}
	for _, tc := range []struct{ from, to string }{{"draft", "review"}, {"active", "archived"}, {"archived", "draft"}} {
		svc, _ := activateSvc("single", tc.from, blank)
		_, err := svc.TransitionStatus(context.Background(), "q-1", tc.to)
		require.NoError(t, err, tc.from+"->"+tc.to)
	}
}

func TestQHandlerTransitionStatus_BlankOptions_Returns422WithFields(t *testing.T) {
	svc := &mockQService{
		transitionStatusFn: func(_ context.Context, _, _ string) (*Question, error) {
			return nil, fmt.Errorf("wrap: %w", &OptionValidationError{Fields: []fieldError{
				{Field: "answer_options[1].translations.kk.text", Message: "blank"},
			}})
		},
	}
	h := NewHandler(svc, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/questions/q-1/status", strings.NewReader(`{"status":"active"}`))
	req = withQChiParam(req, "id", "q-1")
	w := httptest.NewRecorder()
	h.TransitionStatus(w, req)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), `"ERR_VALIDATION"`)
	assert.Contains(t, w.Body.String(), "answer_options[1].translations.kk.text")
}
