import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'

export function EmptyResults() {
  const { t } = useTranslation()
  return (
    <div className="flex flex-col items-center justify-center py-20 text-center gap-4">
      <p className="text-muted-foreground text-lg">{t('history.empty')}</p>
      <Link to="/portal" className="text-primary hover:underline font-medium">
        {t('history.empty_cta')}
      </Link>
    </div>
  )
}
