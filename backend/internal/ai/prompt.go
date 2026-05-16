package ai

import (
	"bytes"
	"fmt"
	"text/template"
)

// insightPromptTemplate is the prompt sent to the Anthropic model for performance insight summaries.
// The mul FuncMap function multiplies a float64 by 100 so percentages render correctly.
const insightPromptTemplate = `
You are an expert HR and learning analytics consultant reviewing exam performance data.
Analyse the following aggregate statistics for the exam "{{.ExamTitle}}" and generate
3 to 5 concise, specific, actionable insights for the exam administrator.

IMPORTANT: Do not mention any individual employees. Only discuss aggregate patterns.

Exam Statistics:
- Total attempts: {{.TotalAttempts}}
- Pass rate: {{printf "%.1f" (mul .PassRate 100)}}%
- Passing score threshold: {{.PassingScorePct}}%
- Average score: {{printf "%.1f" (mul .AvgScorePct 100)}}%
- Average completion time: {{.AvgCompletionSecs}} seconds

Per-Question Performance (Question # — Correct Rate — Stem excerpt):
{{range .QuestionStats}}
  Q{{.OrderNum}} ({{printf "%.0f" (mul .CorrectRate 100)}}% correct, avg {{.AvgTimeSecs}}s): {{.Stem}}
{{end}}

Return ONLY a JSON array of insight strings, no markdown:
["insight 1", "insight 2", ...]
`

// BuildInsightPrompt renders the insight prompt template with the given exam data.
func BuildInsightPrompt(data ExamInsightData) (string, error) {
	funcMap := template.FuncMap{
		"mul": func(a, b float64) float64 { return a * b },
	}
	tmpl, err := template.New("insight").Funcs(funcMap).Parse(insightPromptTemplate)
	if err != nil {
		return "", fmt.Errorf("ai: build insight prompt: parse template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("ai: build insight prompt: execute template: %w", err)
	}
	return buf.String(), nil
}

// generateQuestionsPromptTemplate is the prompt sent to the Anthropic model for question generation.
// It instructs the model to return only valid JSON matching GeneratedQuestionsResponse.
const generateQuestionsPromptTemplate = `
You are an expert exam question writer for corporate training platforms.
Generate {{.Count}} exam questions for the category "{{.CategoryName}}" at {{.Difficulty}} difficulty level.
{{if .ContextText}}Use the following context to inform the questions:
{{.ContextText}}{{end}}

Return ONLY valid JSON with no markdown, matching exactly this schema:
{
  "questions": [
    {
      "type": "single_choice | multiple_choice | true_false",
      "difficulty": "{{.Difficulty}}",
      "stem": "string",
      "explanation": "string (why the correct answer is correct)",
      "options": [{"text": "string", "is_correct": bool}],
      "tags": ["string"]
    }
  ]
}

Requirements:
- Each single_choice question has exactly 4 options with exactly 1 correct
- Each multiple_choice question has 4-6 options with 2-3 correct
- Each true_false question has exactly 2 options (True, False)
- Explanations are 1-3 sentences
- Questions are clear, unambiguous, and professionally worded
`

// ── FR-BB75: Loyalty Profile Narrative ───────────────────────────────────────

// loyaltyNarrativePromptTemplate is the prompt sent to the Anthropic model for
// loyalty profile narratives. No employee names or IDs are included — only
// anonymised, polarity-normalised Likert response weights.
const loyaltyNarrativePromptTemplate = `You are an organizational psychologist interpreting Likert-scale survey responses from a corporate values assessment.

Below are the anonymized response patterns from a single assessment. Each line shows:
  Dimension | Normalized Weight (1=Low alignment, 5=High alignment — polarity already normalized)
{{range .Responses}}
- {{.DimensionLabel}} | NormalizedWeight: {{.NormalizedWeight}}
{{end}}
Write a 2-3 sentence narrative (third person, professional tone) describing the values profile these responses suggest. Do NOT mention any individual by name or pronoun. Use language like "the responses indicate" or "this profile suggests". Do not diagnose, make medical claims, or render employment judgments. Focus only on workplace values and organizational alignment patterns.
`

// BuildLoyaltyPrompt renders the loyalty narrative prompt template with the given data.
func BuildLoyaltyPrompt(data LoyaltyPromptData) (string, error) {
	tmpl, err := template.New("loyalty").Parse(loyaltyNarrativePromptTemplate)
	if err != nil {
		return "", fmt.Errorf("ai: build loyalty prompt: parse template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("ai: build loyalty prompt: execute template: %w", err)
	}
	return buf.String(), nil
}
