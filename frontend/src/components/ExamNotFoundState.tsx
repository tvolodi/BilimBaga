import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ArrowLeft } from 'lucide-react'

/**
 * #470: an exam id that does not exist gets its own message and a way back to the exam list,
 * instead of a loading state or the generic load error.
 */
export function ExamNotFoundState() {
  const { t } = useTranslation()

  return (
    <div role="alert" className="rounded-lg border bg-muted/30 p-12 text-center space-y-4">
      <p className="text-muted-foreground">{t('common.examNotFound')}</p>
      <Link
        to="/admin/exams"
        className="inline-flex items-center gap-1 text-sm text-primary hover:underline"
      >
        <ArrowLeft size={16} aria-hidden="true" />
        {t('nav.exams')}
      </Link>
    </div>
  )
}
