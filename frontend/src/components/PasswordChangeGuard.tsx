import { useEffect } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import type { CurrentUser } from '@/api/auth'
import { PASSWORD_CHANGE_FLAG_KEY } from '@/lib/passwordChangeRequired'

/**
 * ISS-160: while the signed-in user must change their password (login response flag, or the
 * backend answered 403 PASSWORD_CHANGE_REQUIRED), keep them on /change-password. Renders nothing.
 */
export function PasswordChangeGuard() {
  const navigate = useNavigate()
  const { pathname } = useLocation()
  const { data: flag } = useQuery<boolean>({
    queryKey: PASSWORD_CHANGE_FLAG_KEY,
    queryFn: () => false,
    enabled: false,
    staleTime: Infinity,
  })
  const { data: user } = useQuery<CurrentUser | null>({
    queryKey: ['auth', 'currentUser'],
    queryFn: () => null,
    enabled: false,
    staleTime: Infinity,
  })
  const mustChange = flag === true || user?.force_password_change === true

  useEffect(() => {
    if (mustChange && pathname !== '/change-password') {
      navigate('/change-password', { replace: true })
    }
  }, [mustChange, pathname, navigate])

  return null
}
