package router_test

import (
	"os"
	"strings"
	"testing"
)

// #519, architect fix list item 2: the three AI routes share one group with the write-deadline middleware. The test reads
// router.go: the group starts with g.Use(ai.WriteDeadline(ai.AIWriteBudget)), and every AI route sits after that call.
func TestRouter_AIRoutesShareTheWriteDeadlineGroup(t *testing.T) {
	raw, err := os.ReadFile("router.go")
	if err != nil {
		t.Fatalf("read router.go: %v", err)
	}
	src := string(raw)
	group := strings.Index(src, "g.Use(ai.WriteDeadline(ai.AIWriteBudget))")
	if group < 0 {
		t.Fatal("the AI routes must start with g.Use(ai.WriteDeadline(ai.AIWriteBudget))")
	}
	for _, route := range []string{
		`"/admin/ai/generate-questions"`,
		`"/admin/ai/insights/{examId}"`,
		`"/admin/ai/loyalty-summary/{sessionId}"`,
	} {
		at := strings.Index(src, route)
		if at < 0 {
			t.Errorf("route %s not found in router.go", route)
			continue
		}
		if at < group {
			t.Errorf("route %s is not inside the write-deadline group", route)
		}
	}
}
