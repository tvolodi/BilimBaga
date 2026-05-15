import { NavLink, Outlet } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'

export function PortalLayout() {
  const { t } = useTranslation()

  const tabClass = ({ isActive }: { isActive: boolean }) =>
    cn(
      'px-4 py-2 text-sm font-medium border-b-2 transition-colors',
      isActive
        ? 'border-primary text-primary'
        : 'border-transparent text-muted-foreground hover:text-foreground hover:border-muted-foreground',
    )

  return (
    <div className="min-h-screen bg-background">
      <div className="border-b bg-background sticky top-0 z-10">
        <nav className="max-w-7xl mx-auto flex px-6" aria-label="Portal navigation">
          <NavLink to="/portal" end className={tabClass}>
            {t('portal.tab_exams', 'My Exams')}
          </NavLink>
          <NavLink to="/portal/results" className={tabClass}>
            {t('portal.tab_results', 'My Results')}
          </NavLink>
        </nav>
      </div>
      <Outlet />
    </div>
  )
}
