package exams

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ── NextDifficulty tests (FR-BB72 AC-4) ─────────────────────────────────────

func TestNextDifficulty_LessThan3Results_MaintainsCurrent(t *testing.T) {
	assert.Equal(t, Medium, NextDifficulty(Medium, []bool{}))
	assert.Equal(t, Medium, NextDifficulty(Medium, []bool{true}))
	assert.Equal(t, Medium, NextDifficulty(Medium, []bool{true, false}))
	assert.Equal(t, Easy, NextDifficulty(Easy, []bool{true, true}))
	assert.Equal(t, Hard, NextDifficulty(Hard, []bool{false}))
}

func TestNextDifficulty_AllCorrect_IncreasesLevel(t *testing.T) {
	// 3 correct from Easy → Medium
	result := NextDifficulty(Easy, []bool{true, true, true})
	assert.Equal(t, Medium, result)

	// 3 correct from Medium → Hard
	result = NextDifficulty(Medium, []bool{true, true, true})
	assert.Equal(t, Hard, result)
}

func TestNextDifficulty_AllCorrect_CappedAtHard(t *testing.T) {
	// 3 correct from Hard → stays Hard (ceiling)
	result := NextDifficulty(Hard, []bool{true, true, true})
	assert.Equal(t, Hard, result)
}

func TestNextDifficulty_AllWrong_DecreasesLevel(t *testing.T) {
	// 3 wrong from Hard → Medium
	result := NextDifficulty(Hard, []bool{false, false, false})
	assert.Equal(t, Medium, result)

	// 3 wrong from Medium → Easy
	result = NextDifficulty(Medium, []bool{false, false, false})
	assert.Equal(t, Easy, result)
}

func TestNextDifficulty_AllWrong_FlooredAtEasy(t *testing.T) {
	// 3 wrong from Easy → stays Easy (floor)
	result := NextDifficulty(Easy, []bool{false, false, false})
	assert.Equal(t, Easy, result)
}

func TestNextDifficulty_MixedResults_Maintains(t *testing.T) {
	// 1 correct out of 3 (neither >2 nor <1) → maintain
	result := NextDifficulty(Medium, []bool{true, false, false})
	assert.Equal(t, Medium, result)

	result = NextDifficulty(Easy, []bool{false, true, false})
	assert.Equal(t, Easy, result)
}

func TestNextDifficulty_OnlyLast3Matter(t *testing.T) {
	// First 10 are all wrong, but last 3 are all correct → should increase
	results := make([]bool, 13)
	for i := 0; i < 10; i++ {
		results[i] = false
	}
	results[10] = true
	results[11] = true
	results[12] = true

	result := NextDifficulty(Easy, results)
	assert.Equal(t, Medium, result)
}

func TestNextDifficulty_ExactlyTwoCorrect_Maintains(t *testing.T) {
	// 2 correct (not >2) → maintain
	result := NextDifficulty(Medium, []bool{true, true, false})
	assert.Equal(t, Medium, result)
}

func TestNextDifficulty_UnknownDifficulty_DefaultsMedium(t *testing.T) {
	// Unknown difficulty → treated as medium index
	result := NextDifficulty("unknown", []bool{true, true, true})
	// Medium (index 1) + 1 = Hard (index 2)
	assert.Equal(t, Hard, result)
}
