package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// #439: the recovery reset stamps password_changed_at at the start of the next second, like the
// admin reset. A token issued in the reset's own second is then rejected by the access-token epoch.
// The recovery repository's SQL is not reachable from the fake repository, so the statement itself
// is pinned here. The behaviour needs a real database and is checked by UAT's probe (#439 close rule).
func TestRecoveryReset_StampsNextSecond(t *testing.T) {
	assert.Contains(t, completeResetSetPasswordSQL, "password_changed_at = date_trunc('second', now()) + interval '1 second'")
}
