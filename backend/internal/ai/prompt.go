package ai

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
