package sessions

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// FR-BB73 AC-13: the suggest branch neither finalises the session nor sends a mail, whatever calls it makes. The guard
// reads the body of applyAISuggestion and fails on any reference to the finalise helper, the post-commit callback, or a
// mail trigger.
func TestAIGrade_SuggestBranchHasNoFinaliseOrMail(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "aigradejob.go", nil, 0)
	require.NoError(t, err)

	var body *ast.BlockStmt
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "applyAISuggestion" {
			body = fn.Body
		}
	}
	require.NotNil(t, body, "applyAISuggestion must exist in aigradejob.go")

	forbidden := []string{"finaliseSessionIfAllGraded", "OnFinalised", "TriggerSessionResult", "TriggerExamSubmitted", "TriggerResultCorrected", "emailSvc", "mailer"}
	ast.Inspect(body, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.Ident:
			for _, name := range forbidden {
				if v.Name == name {
					t.Errorf("applyAISuggestion refers to %s: the suggest branch must not finalise or mail", name)
				}
			}
		case *ast.SelectorExpr:
			for _, name := range forbidden {
				if strings.EqualFold(v.Sel.Name, name) {
					t.Errorf("applyAISuggestion calls %s: the suggest branch must not finalise or mail", name)
				}
			}
		}
		return true
	})
}
