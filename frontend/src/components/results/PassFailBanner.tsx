import { useTranslation } from 'react-i18next'

interface PassFailBannerProps {
  passed: boolean
}

export function PassFailBanner({ passed }: PassFailBannerProps) {
  const { t } = useTranslation()

  if (passed) {
    return (
      <div className="w-full rounded-lg border border-green-500 bg-green-50 px-4 py-3 text-center text-lg font-semibold text-green-800">
        {t('result.passed')}
      </div>
    )
  }

  return (
    <div className="w-full rounded-lg border border-red-500 bg-red-50 px-4 py-3 text-center text-lg font-semibold text-red-800">
      {t('result.failed')}
    </div>
  )
}
