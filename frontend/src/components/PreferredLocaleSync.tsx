import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useQuery } from '@tanstack/react-query'
import { useMe } from '@/api/users'
import { applyLocale, isSupportedLocale } from '@/lib/locale'

/**
 * FR-BB116 AC-7: once a session exists (after login, or on bootstrap with a refreshed token) the
 * persisted preferred_locale becomes the UI language. A null preference leaves the localStorage /
 * tenant default behaviour unchanged. Renders nothing.
 */
export function PreferredLocaleSync() {
  const { i18n } = useTranslation()
  // Read-only observer of the token that useRefreshToken / useLogin own; it is never fetched here.
  const { data: token } = useQuery<string | null>({
    queryKey: ['auth', 'accessToken'],
    queryFn: () => null,
    enabled: false,
  })
  const me = useMe({ enabled: !!token })
  const preferred = me.data?.preferred_locale ?? null

  useEffect(() => {
    if (isSupportedLocale(preferred) && preferred !== i18n.language) {
      applyLocale(i18n, preferred)
    }
  }, [preferred, i18n])

  return null
}
