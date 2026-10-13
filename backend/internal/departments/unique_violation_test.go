package departments

import (
	"errors"
	"fmt"
	"testing"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestIsUniqueViolation(t *testing.T) {
	assert.True(t, isUniqueViolation(&pq.Error{Code: "23505"}))
	assert.True(t, isUniqueViolation(fmt.Errorf("wrapped: %w", &pq.Error{Code: "23505"})))
	assert.False(t, isUniqueViolation(&pq.Error{Code: "23503"}))
	assert.False(t, isUniqueViolation(errors.New("plain")))
	assert.False(t, isUniqueViolation(nil))
}
