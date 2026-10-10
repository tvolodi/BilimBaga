import { useTranslation } from 'react-i18next'

interface PassFailBannerProps {
  passed: boolean
}

export function PassFailBanner({ passed }: PassFailBannerProps) {
  const { t } = useTranslation()

  if (passed) {
    return (
      <div data-testid="pass-fail-banner" className="w-full rounded-lg border border-success bg-bg-success px-4 py-3 text-center text-lg font-semibold text-success">
        {t('result.passed')}
      </div>
    )
  }

  return (
    <div data-testid="pass-fail-banner" className="w-full rounded-lg border border-danger bg-bg-danger px-4 py-3 text-center text-lg font-semibold text-danger">
      {t('result.failed')}
    </div>
  )
}
