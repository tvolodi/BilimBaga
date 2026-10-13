package sessions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGradeQuestion_UnknownTypeIsGradedZero(t *testing.T) {
	score, max, status, err := gradeQuestion("essay", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 0.0, score)
	assert.Equal(t, 1.0, max)
	assert.Equal(t, GradingStatusGraded, status)
}

func TestGradeQuestion_MultipleRoutesAndErrorsWithoutCorrectOption(t *testing.T) {
	opts := []gradingOptionRow{{ID: "a", IsCorrect: true}, {ID: "b", IsCorrect: true}, {ID: "c"}}
	score, max, status, err := gradeQuestion("multiple", opts, []string{"a", "c"})
	require.NoError(t, err)
	assert.Equal(t, 0.0, score) // 1 correct - 1 incorrect, clamped to 0
	assert.Equal(t, 1.0, max)
	assert.Equal(t, GradingStatusGraded, status)

	_, _, _, err = gradeQuestion("multiple", []gradingOptionRow{{ID: "c"}}, nil)
	assert.Error(t, err)
	_, _, _, err = gradeQuestion("single", []gradingOptionRow{{ID: "c"}}, []string{"c"})
	assert.Error(t, err)
}
