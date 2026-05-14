import { useTranslation } from 'react-i18next'

export function DepartmentsPage() {
  const { t } = useTranslation()
  return (
    <div className="flex flex-col items-center justify-center py-24 text-center">
      <h2 className="text-2xl font-semibold text-gray-700">{t('nav.departments')}</h2>
      <p className="mt-2 text-gray-500">{t('common.coming_soon')}</p>
    </div>
  )
}
