package exams

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseIntParam_Table(t *testing.T) {
	cases := []struct {
		name, query string
		def, want   int
	}{
		{"missing uses default", "", 7, 7},
		{"valid value", "page=3", 1, 3},
		{"non numeric uses default", "page=abc", 1, 1},
		{"zero uses default", "page=0", 5, 5},
		{"negative uses default", "page=-2", 5, 5},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/exams?"+c.query, nil)
			assert.Equal(t, c.want, parseIntParam(r, "page", c.def))
		})
	}
}

func TestPublishValidationError_ErrorAndUnwrap(t *testing.T) {
	var err error = &PublishValidationError{Details: []RuleUnsatisfiedDetail{{}}}
	assert.Equal(t, ErrRulesUnsatisfied.Error(), err.Error())
	assert.True(t, errors.Is(err, ErrRulesUnsatisfied))
}

func TestAdaptivePublishValidationError_ErrorAndUnwrap(t *testing.T) {
	var err error = &AdaptivePublishValidationError{RuleID: "r1", Difficulty: "hard", Required: 3, Available: 1}
	assert.Equal(t, ErrInsufficientAdaptiveQuestions.Error(), err.Error())
	assert.True(t, errors.Is(err, ErrInsufficientAdaptiveQuestions))
}

type deleteSectionErrSvc struct {
	mockSvc
	err error
}

func (s *deleteSectionErrSvc) DeleteSection(context.Context, string, string) error { return s.err }

func TestDeleteSection_ErrorMapping(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code int
	}{
		{"not found", ErrNotFound, http.StatusNotFound},
		{"internal", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := newHandler(&deleteSectionErrSvc{err: c.err})
			w := httptest.NewRecorder()
			req := withChiParam(
				httptest.NewRequest(http.MethodDelete, "/api/v1/exams/exam-1/sections/sec-1", nil),
				"id", "exam-1", "sectionId", "sec-1",
			)
			h.DeleteSection(w, req)
			assert.Equal(t, c.code, w.Code)
		})
	}
}
