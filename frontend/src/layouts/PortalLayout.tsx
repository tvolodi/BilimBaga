import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { LogOut } from 'lucide-react'
import { cn } from '@/lib/utils'
import { LocaleSwitcher } from '@/components/LocaleSwitcher'
import { Button } from '@/components/ui/button'
import { useLogout } from '@/api/auth'

export function PortalLayout() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const logout = useLogout()

  async function handleLogout() {
    await logout.mutateAsync()
    navigate('/login', { replace: true })
  }

  const tabClass = ({ isActive }: { isActive: boolean }) =>
    cn(
      'px-2 sm:px-4 py-2 text-sm font-medium border-b-2 transition-colors text-center',
      isActive
        ? 'border-primary text-primary'
        : 'border-transparent text-muted-foreground hover:text-foreground hover:border-muted-foreground',
    )

  return (
    <div className="min-h-screen bg-background">
      <div className="border-b bg-background sticky top-0 z-10">
        <nav className="max-w-7xl mx-auto flex flex-wrap justify-between items-center gap-x-2 px-2 sm:px-6" aria-label="Portal navigation">
          <div className="flex min-w-0 max-w-full">
            <NavLink to="/portal" end className={tabClass}>
              {t('portal.tab_exams', 'My Exams')}
            </NavLink>
            <NavLink to="/portal/results" className={tabClass}>
              {t('portal.tab_results', 'My Results')}
            </NavLink>
          </div>
          <div className="flex items-center gap-1 sm:gap-3 ml-auto shrink-0">
            <LocaleSwitcher />
            <Button
              variant="ghost"
              size="sm"
              onClick={handleLogout}
              disabled={logout.isPending}
              aria-label={t('common.signOut')}
              className="gap-1.5 text-muted-foreground hover:text-foreground"
            >
              <LogOut size={16} aria-hidden="true" />
            </Button>
          </div>
        </nav>
      </div>
      <main id="main-content" tabIndex={-1} className="outline-none">
        <Outlet />
      </main>
    </div>
  )
}
