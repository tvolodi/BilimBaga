package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteErrorWithDetails(t *testing.T) {
	w := httptest.NewRecorder()
	WriteErrorWithDetails(w, http.StatusConflict, "X", "msg", map[string]interface{}{"count": 3})
	if w.Code != http.StatusConflict {
		t.Fatalf("status %d", w.Code)
	}
	var b struct {
		Data  any `json:"data"`
		Error struct {
			Code    string         `json:"code"`
			Message string         `json:"message"`
			Details map[string]any `json:"details"`
		} `json:"error"`
	}
	if err := json.NewDecoder(w.Body).Decode(&b); err != nil {
		t.Fatal(err)
	}
	if b.Data != nil || b.Error.Code != "X" || b.Error.Message != "msg" || b.Error.Details["count"] != float64(3) {
		t.Fatalf("unexpected body %+v", b)
	}
}
