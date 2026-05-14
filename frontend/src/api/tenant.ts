import { useMutation, useQueryClient } from '@tanstack/react-query'

export interface ApiError {
  code: string
  message: string
  details?: Record<string, string>
}

export interface TenantConfigUpdate {
  app_name?: string
  logo?: string
  primary_color?: string
  accent_color?: string
  default_locale?: string
  available_locales?: string[]
}

interface ApiEnvelope<T> {
  data: T | null
  error: ApiError | null
}

export function useUpdateTenantConfig() {
  const qc = useQueryClient()
  return useMutation<void, ApiError, TenantConfigUpdate>({
    mutationFn: async (payload) => {
      const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
      const res = await fetch('/api/v1/tenant/config', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify(payload),
        credentials: 'include',
      })
      const json: ApiEnvelope<unknown> = await res.json().catch(() => ({ data: null, error: null }))
      if (!res.ok || json.error) {
        throw (json.error ?? { code: 'NETWORK_ERROR', message: `Request failed: ${res.status}` })
      }
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['tenant', 'config'] })
    },
  })
}
