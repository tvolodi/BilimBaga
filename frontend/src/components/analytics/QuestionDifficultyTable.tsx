import { useTranslation } from 'react-i18next'
import { ChevronUp, ChevronDown } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { AnswerDistributionExpander } from './AnswerDistributionExpander'
import type { QuestionStat } from '@/api/analytics'

type SortCol = 'correct_rate' | 'avg_time_seconds'
type SortDir = 'asc' | 'desc'

interface QuestionDifficultyTableProps {
  questions: QuestionStat[]
  sortCol: SortCol
  sortDir: SortDir
  onSort: (col: SortCol) => void
}

function SortIcon({
  col,
  activeCol,
  dir,
}: {
  col: SortCol
  activeCol: SortCol
  dir: SortDir
}) {
  if (col !== activeCol) {
    return <ChevronDown size={14} className="text-muted-foreground/40" />
  }
  return dir === 'asc' ? (
    <ChevronUp size={14} className="text-foreground" />
  ) : (
    <ChevronDown size={14} className="text-foreground" />
  )
}

function getRowClass(correctRate: number | null): string {
  if (correctRate === null) return ''
  if (correctRate < 0.4) return 'border-l-4 border-l-red-500'
  if (correctRate > 0.95) return 'border-l-4 border-l-amber-400'
  return ''
}

function RevisionBadge({ correctRate }: { correctRate: number | null }) {
  const { t } = useTranslation()

  if (correctRate === null) return null

  if (correctRate < 0.4) {
    return (
      <span
        title={t('exam_analytics.badge_revise')}
        aria-label={t('exam_analytics.badge_revise')}
      >
        <Badge variant="destructive" className="ml-2 cursor-help">
          {t('exam_analytics.badge_revise')}
        </Badge>
      </span>
    )
  }

  if (correctRate > 0.95) {
    return (
      <span
        title={t('exam_analytics.badge_remove')}
        aria-label={t('exam_analytics.badge_remove')}
      >
        <Badge variant="warning" className="ml-2 cursor-help">
          {t('exam_analytics.badge_remove')}
        </Badge>
      </span>
    )
  }

  return null
}

export function QuestionDifficultyTable({
  questions,
  sortCol,
  sortDir,
  onSort,
}: QuestionDifficultyTableProps) {
  const { t } = useTranslation()

  return (
    <div className="overflow-x-auto rounded-lg border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b bg-muted/50">
            <th className="px-4 py-3 text-left font-medium text-muted-foreground w-10">
              {t('exam_analytics.col_number')}
            </th>
            <th className="px-4 py-3 text-left font-medium text-muted-foreground">
              {t('exam_analytics.col_question')}
            </th>
            <th
              className="px-4 py-3 text-left font-medium text-muted-foreground cursor-pointer select-none hover:text-foreground transition-colors"
              onClick={() => onSort('correct_rate')}
            >
              <span className="flex items-center gap-1">
                {t('exam_analytics.col_correct_rate')}
                <SortIcon col="correct_rate" activeCol={sortCol} dir={sortDir} />
              </span>
            </th>
            <th
              className="px-4 py-3 text-left font-medium text-muted-foreground cursor-pointer select-none hover:text-foreground transition-colors"
              onClick={() => onSort('avg_time_seconds')}
            >
              <span className="flex items-center gap-1">
                {t('exam_analytics.col_avg_time')}
                <SortIcon col="avg_time_seconds" activeCol={sortCol} dir={sortDir} />
              </span>
            </th>
          </tr>
        </thead>
        <tbody>
          {questions.map((q, idx) => (
            <tr
              key={q.question_id}
              className={cn(
                'border-b last:border-0 hover:bg-muted/30 transition-colors',
                getRowClass(q.correct_rate),
              )}
            >
              <td className="px-4 py-3 text-muted-foreground">{idx + 1}</td>
              <td className="px-4 py-3">
                <div className="flex flex-wrap items-center gap-1">
                  <span>
                    {q.stem_preview.length > 80
                      ? `${q.stem_preview.slice(0, 80)}…`
                      : q.stem_preview}
                  </span>
                  <RevisionBadge correctRate={q.correct_rate} />
                </div>
                <AnswerDistributionExpander distribution={q.answer_distribution} />
              </td>
              <td className="px-4 py-3">
                {q.correct_rate !== null
                  ? `${(q.correct_rate * 100).toFixed(1)}%`
                  : '—'}
              </td>
              <td className="px-4 py-3">
                {q.avg_time_seconds !== null
                  ? `${q.avg_time_seconds.toFixed(1)}s`
                  : '—'}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
