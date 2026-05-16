package exams

// Difficulty represents the difficulty level of a question used by the adaptive algorithm.
type Difficulty string

const (
	Easy   Difficulty = "easy"
	Medium Difficulty = "medium"
	Hard   Difficulty = "hard"

	// AnyDifficulty is a sentinel used to request a question at any difficulty level.
	AnyDifficulty Difficulty = ""
)

// difficultyOrder defines the ordered scale from easy to hard.
var difficultyOrder = []Difficulty{Easy, Medium, Hard}

// NextDifficulty computes the next difficulty based on the recent result window.
// Rule (FR-BB72 AC-4):
//   - If len(recentResults) < 3: maintain current difficulty (not enough data).
//   - If last 3 correct > 2: increase one level (capped at Hard).
//   - If last 3 correct < 1: decrease one level (floored at Easy).
//   - Otherwise: maintain current difficulty.
func NextDifficulty(current Difficulty, recentResults []bool) Difficulty {
	if len(recentResults) < 3 {
		return current // not enough data yet, maintain
	}
	last3 := recentResults[len(recentResults)-3:]
	correct := 0
	for _, r := range last3 {
		if r {
			correct++
		}
	}
	idx := difficultyIndex(current)
	if correct > 2 && idx < len(difficultyOrder)-1 {
		return difficultyOrder[idx+1]
	}
	if correct < 1 && idx > 0 {
		return difficultyOrder[idx-1]
	}
	return current
}

// difficultyIndex returns the index of d in difficultyOrder.
// Returns 1 (Medium) for unknown difficulties.
func difficultyIndex(d Difficulty) int {
	for i, v := range difficultyOrder {
		if v == d {
			return i
		}
	}
	return 1 // default medium
}
