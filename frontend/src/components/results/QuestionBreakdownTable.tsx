import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import type { QuestionBreakdownItem } from '@/api/sessions'

interface QuestionBreakdownTableProps {
  breakdown: QuestionBreakdownItem[] | undefined
}

const STEM_MAX = 120
const EXPLANATION_MAX = 200

function ExplanationCell({ text }: { text: string | null }) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(false)

  if (!text) return <span className="text-muted-foreground">—</span>

  if (text.length <= EXPLANATION_MAX) {
    return <span>{text}</span>
  }

  return (
    <span>
      {expanded ? text : `${text.slice(0, EXPLANATION_MAX)}…`}{' '}
      <button
        type="button"
        className="ml-1 text-xs underline text-muted-foreground hover:text-foreground"
        onClick={() => setExpanded((v) => !v)}
      >
        {expanded ? t('result.show_less') : t('result.show_more')}
      </button>
    </span>
  )
}

export function QuestionBreakdownTable({ breakdown }: QuestionBreakdownTableProps) {
  const { t } = useTranslation()

  if (!breakdown) return null

  return (
    <div className="w-full">
      <h3 className="mb-3 text-sm font-semibold text-muted-foreground uppercase tracking-wide">
        {t('result.question_review')}
      </h3>
      <div className="rounded-md border overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-[30%]">{t('result.question_stem')}</TableHead>
              <TableHead>{t('result.your_answer')}</TableHead>
              <TableHead>{t('result.correct_answer')}</TableHead>
              <TableHead className="text-right">{t('result.points')}</TableHead>
              <TableHead>{t('result.explanation')}</TableHead>
              <TableHead>{t('result.examiner_feedback')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {breakdown.map((item) => {
              // FR-BB322 AC-8: a question awaiting grading has no points and no right or wrong colour yet.
              const pending = item.grading_status === 'pending_manual'
              const isCorrect =
                item.points_earned > 0 && item.points_earned === item.max_points
              const isWrong = item.points_earned < item.max_points
              const stem =
                item.stem.length > STEM_MAX
                  ? `${item.stem.slice(0, STEM_MAX)}…`
                  : item.stem

              return (
                <TableRow key={item.question_id}>
                  <TableCell className="align-top text-sm">{stem}</TableCell>
                  <TableCell
                    className={`align-top text-sm ${isWrong && !isCorrect && !pending ? 'bg-bg-danger' : ''}`}
                  >
                    {item.employee_answer.join(', ') || '—'}
                  </TableCell>
                  <TableCell
                    className={`align-top text-sm font-semibold ${pending ? '' : 'bg-bg-success'}`}
                  >
                    {pending ? '—' : item.correct_answer.join(', ') || '—'}
                  </TableCell>
                  <TableCell className="align-top text-sm text-right whitespace-nowrap">
                    {pending
                      ? <span className="text-warning">{t('result.question_pending')}</span>
                      : <>{item.points_earned} / {item.max_points}</>}
                  </TableCell>
                  <TableCell className="align-top text-sm max-w-[240px]">
                    <ExplanationCell text={pending ? null : item.explanation} />
                  </TableCell>
                  <TableCell className="align-top text-sm max-w-[240px]">
                    {item.manual_feedback
                      ? <span className="text-info">{item.manual_feedback}</span>
                      : <span className="text-muted-foreground">—</span>}
                  </TableCell>
                </TableRow>
              )
            })}
          </TableBody>
        </Table>
      </div>
    </div>
  )
}
