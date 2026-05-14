import { useTranslation } from 'react-i18next'

export function TenantLogo() {
  const { t } = useTranslation()

  function handleError(e: React.SyntheticEvent<HTMLImageElement>) {
    const img = e.currentTarget
    img.style.display = 'none'
    const next = img.nextElementSibling as HTMLElement | null
    if (next) {
      next.style.display = ''
    }
  }

  return (
    <>
      <img
        src="/api/v1/tenant/logo"
        alt={t('common.logo_placeholder')}
        onError={handleError}
      />
      <span style={{ display: 'none' }}>{t('common.logo_placeholder')}</span>
    </>
  )
}
