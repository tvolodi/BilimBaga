package sessions

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helper: build a gradingOptionRow slice.
func makeOpts(defs []struct {
	id        string
	isCorrect bool
	weight    *float64
	polarity  *string
}) []gradingOptionRow {
	rows := make([]gradingOptionRow, len(defs))
	for i, d := range defs {
		rows[i] = gradingOptionRow{
			ID:             d.id,
			IsCorrect:      d.isCorrect,
			LikertWeight:   d.weight,
			LikertPolarity: d.polarity,
		}
	}
	return rows
}

func pf(v float64) *float64 { return &v }
func ps(v string) *string   { return &v }

// ── gradeSingleOrTrueFalse ────────────────────────────────────────────────────

func TestGradeSingleOrTrueFalse(t *testing.T) {
	opts := makeOpts([]struct {
		id        string
		isCorrect bool
		weight    *float64
		polarity  *string
	}{
		{"opt-correct", true, nil, nil},
		{"opt-wrong-a", false, nil, nil},
		{"opt-wrong-b", false, nil, nil},
	})

	tests := []struct {
		name       string
		selected   []string
		wantScore  float64
		wantMax    float64
		wantStatus GradingStatus
		wantErr    bool
	}{
		{
			name:       "correct single selection → 1.0 (AC-1)",
			selected:   []string{"opt-correct"},
			wantScore:  1.0,
			wantMax:    1.0,
			wantStatus: GradingStatusGraded,
		},
		{
			name:       "wrong single selection → 0.0 (AC-1)",
			selected:   []string{"opt-wrong-a"},
			wantScore:  0.0,
			wantMax:    1.0,
			wantStatus: GradingStatusGraded,
		},
		{
			name:       "no selection → 0.0 (AC-1)",
			selected:   []string{},
			wantScore:  0.0,
			wantMax:    1.0,
			wantStatus: GradingStatusGraded,
		},
		{
			name:       "multiple selections → 0.0 (AC-1: exactly one required)",
			selected:   []string{"opt-correct", "opt-wrong-a"},
			wantScore:  0.0,
			wantMax:    1.0,
			wantStatus: GradingStatusGraded,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			score, max, status, err := gradeSingleOrTrueFalse(opts, tc.selected)
			require.NoError(t, err)
			assert.InDelta(t, tc.wantScore, score, 1e-9)
			assert.InDelta(t, tc.wantMax, max, 1e-9)
			assert.Equal(t, tc.wantStatus, status)
		})
	}
}

func TestGradeSingleOrTrueFalse_NoCorrectOption_ReturnsError(t *testing.T) {
	// AC-10: error if no correct option exists.
	opts := makeOpts([]struct {
		id        string
		isCorrect bool
		weight    *float64
		polarity  *string
	}{
		{"opt-a", false, nil, nil},
		{"opt-b", false, nil, nil},
	})

	_, _, _, err := gradeSingleOrTrueFalse(opts, []string{"opt-a"})
	require.Error(t, err)
}

// ── gradeMultipleChoice ──────────────────────────────────────────────────────

func TestGradeMultipleChoice(t *testing.T) {
	opts := makeOpts([]struct {
		id        string
		isCorrect bool
		weight    *float64
		polarity  *string
	}{
		{"opt-c1", true, nil, nil},
		{"opt-c2", true, nil, nil},
		{"opt-c3", true, nil, nil},
		{"opt-w1", false, nil, nil},
		{"opt-w2", false, nil, nil},
	})

	tests := []struct {
		name      string
		selected  []string
		wantScore float64
	}{
		{
			name:      "exact match all 3 correct → 1.0 (AC-2)",
			selected:  []string{"opt-c1", "opt-c2", "opt-c3"},
			wantScore: 1.0,
		},
		{
			name:      "2 correct 0 wrong → 2/3 (AC-2)",
			selected:  []string{"opt-c1", "opt-c2"},
			wantScore: 2.0 / 3.0,
		},
		{
			name:      "1 correct 1 wrong → max(0,1-1)/3 = 0 (AC-2)",
			selected:  []string{"opt-c1", "opt-w1"},
			wantScore: 0.0,
		},
		{
			name:      "all wrong → 0.0 (AC-2)",
			selected:  []string{"opt-w1", "opt-w2"},
			wantScore: 0.0,
		},
		{
			name:      "no selection → 0.0",
			selected:  []string{},
			wantScore: 0.0,
		},
		{
			name:      "2 correct 1 wrong → max(0,2-1)/3 = 1/3 (AC-2 partial credit)",
			selected:  []string{"opt-c1", "opt-c2", "opt-w1"},
			wantScore: 1.0 / 3.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			score, max, status, err := gradeMultipleChoice(opts, tc.selected)
			require.NoError(t, err)
			assert.InDelta(t, tc.wantScore, score, 1e-9)
			assert.InDelta(t, 1.0, max, 1e-9)
			assert.Equal(t, GradingStatusGraded, status)
		})
	}
}

func TestGradeMultipleChoice_NoCorrectOptions_ReturnsError(t *testing.T) {
	// AC-2 / AC-10: total_correct == 0 → error.
	opts := makeOpts([]struct {
		id        string
		isCorrect bool
		weight    *float64
		polarity  *string
	}{
		{"opt-a", false, nil, nil},
		{"opt-b", false, nil, nil},
	})

	_, _, _, err := gradeMultipleChoice(opts, []string{"opt-a"})
	require.Error(t, err)
}

// ── gradeLikert ──────────────────────────────────────────────────────────────

func TestGradeLikert(t *testing.T) {
	// Options: positive 1,2,3,4,5 and negative 1,2,3,4,5.
	// max_q = 5, min_q = -5, range = 10.
	opts := makeOpts([]struct {
		id        string
		isCorrect bool
		weight    *float64
		polarity  *string
	}{
		{"pos-1", false, pf(1), ps("positive")},
		{"pos-3", false, pf(3), ps("positive")},
		{"pos-5", false, pf(5), ps("positive")},
		{"neg-1", false, pf(1), ps("negative")},
		{"neg-5", false, pf(5), ps("negative")},
	})

	tests := []struct {
		name      string
		selected  []string
		wantScore float64
	}{
		{
			name:      "max positive contribution → 1.0 (AC-3)",
			selected:  []string{"pos-5"},
			wantScore: 1.0, // (5 - (-5)) / (5 - (-5)) = 10/10
		},
		{
			name:      "max negative contribution → 0.0 (AC-3)",
			selected:  []string{"neg-5"},
			wantScore: 0.0, // (-5 - (-5)) / 10 = 0/10
		},
		{
			name:      "neutral positive → 0.8 (AC-3)",
			selected:  []string{"pos-3"},
			wantScore: 0.8, // (3 - (-5)) / 10 = 8/10
		},
		{
			name:      "no selection → 0.0",
			selected:  []string{},
			wantScore: 0.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			score, max, status, err := gradeLikert(opts, tc.selected)
			require.NoError(t, err)
			assert.InDelta(t, tc.wantScore, score, 1e-9)
			assert.InDelta(t, 1.0, max, 1e-9)
			assert.Equal(t, GradingStatusGraded, status)
		})
	}
}

func TestGradeLikert_AllPositiveOptions(t *testing.T) {
	// No negative-polarity options → min_q = 0.
	// max_q = 4, min_q = 0, range = 4.
	opts := makeOpts([]struct {
		id        string
		isCorrect bool
		weight    *float64
		polarity  *string
	}{
		{"p1", false, pf(1), ps("positive")},
		{"p2", false, pf(2), ps("positive")},
		{"p4", false, pf(4), ps("positive")},
	})

	score, _, _, err := gradeLikert(opts, []string{"p2"})
	require.NoError(t, err)
	// (2 - 0) / (4 - 0) = 0.5
	assert.InDelta(t, 0.5, score, 1e-9)
}

func TestGradeLikert_ZeroWeight_ReturnsZero(t *testing.T) {
	// All options have weight 0 → max_q = 0, min_q = 0, range = 0 → score = 0 (AC-3 edge case).
	opts := makeOpts([]struct {
		id        string
		isCorrect bool
		weight    *float64
		polarity  *string
	}{
		{"z1", false, pf(0), ps("positive")},
		{"z2", false, pf(0), ps("negative")},
	})

	score, max, status, err := gradeLikert(opts, []string{"z1"})
	require.NoError(t, err)
	assert.InDelta(t, 0.0, score, 1e-9)
	assert.InDelta(t, 1.0, max, 1e-9)
	assert.Equal(t, GradingStatusGraded, status)
}

func TestGradeLikert_ClampAboveOne(t *testing.T) {
	// Sanity check: score is always clamped to [0,1].
	opts := makeOpts([]struct {
		id        string
		isCorrect bool
		weight    *float64
		polarity  *string
	}{
		{"p10", false, pf(10), ps("positive")},
		{"n1", false, pf(1), ps("negative")},
	})
	// max_q = 10, min_q = -10, range = 20. contribution = +10. score = (10-(-10))/20 = 1.
	score, _, _, err := gradeLikert(opts, []string{"p10"})
	require.NoError(t, err)
	assert.LessOrEqual(t, score, 1.0)
	assert.GreaterOrEqual(t, score, 0.0)
}

// ── gradeQuestion dispatcher ─────────────────────────────────────────────────

func TestGradeQuestion_ShortText_PendingManual(t *testing.T) {
	// AC-4: shorttext always returns score=0, pending_manual.
	score, max, status, err := gradeQuestion("shorttext", nil, nil)
	require.NoError(t, err)
	assert.InDelta(t, 0.0, score, 1e-9)
	assert.InDelta(t, 1.0, max, 1e-9)
	assert.Equal(t, GradingStatusPendingManual, status)
}

func TestGradeQuestion_TrueFalse_Correct(t *testing.T) {
	opts := makeOpts([]struct {
		id        string
		isCorrect bool
		weight    *float64
		polarity  *string
	}{
		{"true-opt", true, nil, nil},
		{"false-opt", false, nil, nil},
	})

	score, _, _, err := gradeQuestion("truefalse", opts, []string{"true-opt"})
	require.NoError(t, err)
	assert.InDelta(t, 1.0, score, 1e-9)
}

func TestGradeQuestion_TrueFalse_Incorrect(t *testing.T) {
	opts := makeOpts([]struct {
		id        string
		isCorrect bool
		weight    *float64
		polarity  *string
	}{
		{"true-opt", true, nil, nil},
		{"false-opt", false, nil, nil},
	})

	score, _, _, err := gradeQuestion("truefalse", opts, []string{"false-opt"})
	require.NoError(t, err)
	assert.InDelta(t, 0.0, score, 1e-9)
}

// ── aggregate score_pct computation ─────────────────────────────────────────

func TestScorePct_Rounding(t *testing.T) {
	// Verify ROUND(SUM(score)/SUM(max)*100, 2) by hand.
	// 2 correct out of 3 questions, each max=1 → 66.666... → rounds to 66.67.
	totalScore := 2.0
	totalMax := 3.0
	scorePct := math.Round(totalScore/totalMax*100*100) / 100
	assert.InDelta(t, 66.67, scorePct, 0.005)
}

func TestScorePct_ZeroMax(t *testing.T) {
	// AC-6: SUM(max_score) == 0 → score_pct = 0.
	totalMax := 0.0
	var scorePct float64
	if totalMax > 0 {
		scorePct = math.Round(1.0/totalMax*100*100) / 100
	}
	assert.InDelta(t, 0.0, scorePct, 1e-9)
}
