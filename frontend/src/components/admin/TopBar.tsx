import { Link, useNavigate } from 'react-router-dom'
import { LogOut } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { RoleBadge } from '@/components/admin/RoleBadge'
import { LocaleSwitcher } from '@/components/LocaleSwitcher'
import { ThemeToggle } from '@/components/ThemeToggle'
import { useLogout } from '@/api/auth'
import type { User } from '@/api/users'

interface TopBarProps {
  user: User
}

export function TopBar({ user }: TopBarProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const logout = useLogout()

  async function handleLogout() {
    await logout.mutateAsync()
    navigate('/login', { replace: true })
  }

  // FR-BB321 AC-6: at 375 px the controls wrap onto a second line instead of overflowing. The toggle keeps its 44 px targets.
  return (
    <header className="flex min-h-14 flex-wrap items-center justify-end gap-x-3 gap-y-2 px-4 py-2 sm:px-6 border-b bg-card shadow-sm flex-shrink-0">
      <div className="flex flex-wrap items-center justify-end gap-x-3 gap-y-2">
        <LocaleSwitcher />
        <ThemeToggle />
        {/* FR-BB116 AC-5: the name opens the caller's profile page. */}
        <Link to="/admin/profile" className="text-sm font-medium text-foreground hover:underline">
          {user.full_name}
        </Link>
        <RoleBadge role={user.role_name} />
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
    </header>
  )
}
