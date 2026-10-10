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

  return (
    <header className="flex items-center justify-between h-14 px-6 border-b bg-white shadow-sm flex-shrink-0">
      <div />
      <div className="flex items-center gap-3">
        <LocaleSwitcher />
        <ThemeToggle />
        {/* FR-BB116 AC-5: the name opens the caller's profile page. */}
        <Link to="/admin/profile" className="text-sm font-medium text-gray-700 hover:text-gray-900 hover:underline">
          {user.full_name}
        </Link>
        <RoleBadge role={user.role_name} />
        <Button
          variant="ghost"
          size="sm"
          onClick={handleLogout}
          disabled={logout.isPending}
          aria-label={t('common.signOut')}
          className="gap-1.5 text-gray-600 hover:text-gray-900"
        >
          <LogOut size={16} aria-hidden="true" />
        </Button>
      </div>
    </header>
  )
}
