# FR-BB57 — Frontend: Per-Exam Analytics

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB57 |
| Phase | 5 — Analytics & Reporting |
| Priority | 2 |
| Status | uat-verified |
| Depends On | FR-BB52 |

## Description
Renders the detailed analytics page for a single exam. Presents a score distribution histogram, a pass rate visualisation, a summary statistics row, and a sortable per-question difficulty table. Rows in the difficulty table are colour-coded to flag questions that may need revision (very low correct rate) or removal (very high correct rate). A CSV export button downloads the raw session results. All data comes from the per-exam analytics API (FR-BB52).

## Acceptance Criteria
- [ ] AC-1: The page is accessible from the admin exam list via a "View Analytics" button/link on each exam row.
- [ ] AC-2: A score distribution histogram is rendered using `recharts BarChart` with 10 fixed buckets on the X-axis (0–10% through 90–100%) and session count on the Y-axis.
- [ ] AC-3: A pass rate visualisation is rendered using `recharts RadialBarChart` or `PieChart` showing passed vs failed proportion; the passed segment uses the tenant's `primary_color`.
- [ ] AC-4: A statistics row displays four values: average score, median score, total attempts, unique participants.
- [ ] AC-5: The per-question difficulty table is sortable by `correct_rate` (default ascending) and `avg_time_seconds`; sorting is client-side since all data is already loaded.
- [ ] AC-6: Table rows where `correct_rate < 0.40` have a red left border and a "Consider Revising" tooltip badge.
- [ ] AC-7: Table rows where `correct_rate > 0.95` have an amber/yellow left border and a "Consider Removing" tooltip badge.
- [ ] AC-8: The "Export CSV" button calls `GET /admin/exams/:id/results/export` and triggers a browser file download.
- [ ] AC-9: The page uses `useQuery` with `queryKey: ['exam-analytics', examId]`; loading state renders skeletons for each section.
- [ ] AC-10: Zero hardcoded user-visible strings; all text uses `useTranslation` i18n keys.

## Technical Specification

### Frontend Components

#### Page: `ExamAnalyticsPage` (`src/pages/admin/ExamAnalyticsPage.tsx`)
- Route: `/admin/exams/:examId/analytics`
- Guarded by `RequireRole(['examiner','hr_admin','super_admin'])`
- Uses `useExamAnalytics(examId)` hook

#### Hook: `useExamAnalytics`
```ts
// src/api/analytics.ts
export function useExamAnalytics(examId: string) {
  return useQuery({
    queryKey: ['exam-analytics', examId],
    queryFn: () => apiGet<ExamAnalytics>(`/admin/exams/${examId}/analytics`),
    staleTime: 5 * 60 * 1000,
  });
}
```

#### Component: `ScoreDistributionChart` (`src/components/analytics/ScoreDistributionChart.tsx`)
```tsx
// Props: { distribution: BucketCount[]; primaryColor: string }
// recharts: <ResponsiveContainer width="100%" height={240}>
//   <BarChart data={distribution}>
//     <CartesianGrid strokeDasharray="3 3" />
//     <XAxis dataKey="bucket" tick={{ fontSize: 11 }} />
//     <YAxis allowDecimals={false} />
//     <Tooltip />
//     <Bar dataKey="count" fill={primaryColor} radius={[4, 4, 0, 0]} />
//   </BarChart>
// </ResponsiveContainer>
```

#### Component: `PassRateChart` (`src/components/analytics/PassRateChart.tsx`)
```tsx
// Props: { passRate: number; primaryColor: string }
// recharts PieChart with two cells: passed (primaryColor) and failed (muted grey)
// Centre label: "{(passRate * 100).toFixed(1)}% Passed"
// Compact size: 200×200
```

#### Component: `StatsSummaryRow` (`src/components/analytics/StatsSummaryRow.tsx`)
```tsx
// Props: { avgScore, medianScore, totalAttempts, uniqueParticipants }
// 4-column grid of shadcn Card components
// Each card: label + large numeric value
```

#### Component: `QuestionDifficultyTable` (`src/components/analytics/QuestionDifficultyTable.tsx`)
```tsx
// Props: { questions: QuestionStat[]; onSort: fn; sortCol: string; sortDir: string }
// Columns: #, Question (truncated 80 chars), Correct Rate (%), Avg Time (s)
// Row styling:
//   correct_rate < 0.40  → className="border-l-4 border-l-red-500"
//   correct_rate > 0.95  → className="border-l-4 border-l-amber-400"
// Shadcn Tooltip on badge: "Consider Revising" / "Consider Removing"
// Sortable headers: clicking toggles asc/desc, shows ChevronUp/ChevronDown icon
```

#### Component: `AnswerDistributionExpander`
```tsx
// Props: { distribution: AnswerDistributionItem[] }
// Collapsible accordion within each table row
// Shows option text + horizontal progress bar (count / total sessions)
```

#### Component: `ExportCSVButton`
```tsx
// Props: { examId: string }
// On click: fetch blob from /admin/exams/:id/results/export → download
// Inline loading state
```

### Client-side Sort Logic

```ts
function sortQuestions(
  questions: QuestionStat[],
  col: 'correct_rate' | 'avg_time_seconds',
  dir: 'asc' | 'desc'
): QuestionStat[] {
  return [...questions].sort((a, b) => {
    const av = a[col] ?? 0;
    const bv = b[col] ?? 0;
    return dir === 'asc' ? av - bv : bv - av;
  });
}
```

### i18n Keys Required

```json
{
  "exam_analytics.title": "Exam Analytics",
  "exam_analytics.score_distribution": "Score Distribution",
  "exam_analytics.pass_rate": "Pass Rate",
  "exam_analytics.avg_score": "Average Score",
  "exam_analytics.median_score": "Median Score",
  "exam_analytics.total_attempts": "Total Attempts",
  "exam_analytics.unique_participants": "Unique Participants",
  "exam_analytics.question_difficulty": "Question Analysis",
  "exam_analytics.col_number": "#",
  "exam_analytics.col_question": "Question",
  "exam_analytics.col_correct_rate": "Correct Rate",
  "exam_analytics.col_avg_time": "Avg Time (s)",
  "exam_analytics.badge_revise": "Consider Revising",
  "exam_analytics.badge_remove": "Consider Removing",
  "exam_analytics.export_csv": "Export Results CSV",
  "exam_analytics.no_data": "No attempts recorded for this exam yet.",
  "exam_analytics.answer_distribution": "Answer Distribution"
}
```

### TypeScript Types

```ts
export interface ExamAnalytics {
  exam_id: string;
  exam_title: string;
  score_distribution: BucketCount[];
  pass_rate: number;
  avg_score: number | null;
  median_score: number | null;
  total_attempts: number;
  unique_participants: number;
  per_question_stats: QuestionStat[];
}

export interface BucketCount {
  bucket: string;
  count: number;
}

export interface QuestionStat {
  question_id: string;
  stem_preview: string;
  correct_rate: number | null;
  avg_time_seconds: number | null;
  answer_distribution: AnswerDistributionItem[];
}

export interface AnswerDistributionItem {
  option_id: string;
  option_text: string;
  select_count: number;
}
```

## Notes
- `recharts` is already listed as a dependency in FR-BB56; both analytics pages share the same library.
- When `total_attempts = 0`, all chart components render an empty state message (i18n key `exam_analytics.no_data`) instead of the chart/table.
- The answer distribution expander is hidden for `short_text` questions (no options).
- Correct rate is displayed as a percentage string `"83.4%"` in the table; sort uses the raw 0–1 decimal value.
