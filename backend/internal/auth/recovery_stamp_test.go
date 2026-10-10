package auth

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

// #455: the recovery reset takes its password_changed_at from NextPasswordStamp (the application clock
// and the previous stamp, read under the row lock), never from the database clock. The statement
// receives the stamp as $3. The behaviour needs a real database; UAT's probe is the close check (#439/#455).
func TestRecoveryReset_StampComesFromTheParameterNotTheDatabaseClock(t *testing.T) {
	assert.Contains(t, completeResetSetPasswordSQL, "password_changed_at = $3")
	stampAssignment := regexp.MustCompile(`password_changed_at\s*=\s*([^\n]*)`).FindStringSubmatch(completeResetSetPasswordSQL)
	if assert.Len(t, stampAssignment, 2) {
		assert.NotContains(t, stampAssignment[1], "now()", "the stamp must not use the database clock")
	}
}
