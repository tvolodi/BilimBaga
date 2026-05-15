import { useTranslation } from 'react-i18next'
import { Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'

export interface GradeMap {
  [questionId: string]: { score_pct: number | null; feedback: string }
}

interface SubmitAllGradesButtonProps {
  grades: GradeMap
  questionCount: number
  onSubmit: () => void
  isLoading: boolean
}

export function SubmitAllGradesButton({
  grades,
  questionCount,
  onSubmit,
  isLoading,
}: SubmitAllGradesButtonProps) {
  const { t } = useTranslation()

  const allAssigned =
    Object.keys(grades).length === questionCount &&
    Object.values(grades).every((g) => g.score_pct !== null && g.score_pct >= 0 && g.score_pct <= 100)

  return (
    <Button onClick={onSubmit} disabled={!allAssigned || isLoading}>
      {isLoading ? (
        <>
          <Loader2 size={16} className="mr-2 animate-spin" />
          {t('grading.submitting')}
        </>
      ) : (
        t('grading.submit_all')
      )}
    </Button>
  )
}
