import { useTranslation } from 'react-i18next'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { Button } from '@/components/ui/button'

interface GradingNavigationProps {
  current: number
  total: number
  onPrev: () => void
  onNext: () => void
}

export function GradingNavigation({ current, total, onPrev, onNext }: GradingNavigationProps) {
  const { t } = useTranslation()

  return (
    <div className="flex items-center justify-between">
      <Button variant="outline" onClick={onPrev} disabled={current <= 1}>
        <ChevronLeft size={16} className="mr-1" />
        {t('grading.prev_question')}
      </Button>
      <span className="text-sm text-muted-foreground">
        {t('grading.question_indicator', { current, total })}
      </span>
      <Button variant="outline" onClick={onNext} disabled={current >= total}>
        {t('grading.next_question')}
        <ChevronRight size={16} className="ml-1" />
      </Button>
    </div>
  )
}
