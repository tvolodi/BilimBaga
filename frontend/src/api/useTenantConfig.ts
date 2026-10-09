import { useQuery } from '@tanstack/react-query'
import { errorWithCode } from '@/api/errors'

export interface TenantConfig {
  app_name: string
  primary_color: string
  accent_color: string
  default_locale: string
  available_locales: string[]
}

interface ApiResponse<T> {
  data: T
  error: null | { code: string; message: string }
}

async function fetchTenantConfig(): Promise<TenantConfig> {
  const res = await fetch('/api/v1/tenant/config')
  if (!res.ok) {
    throw new Error(`Failed to fetch tenant config: ${res.status}`)
  }
  const body: ApiResponse<TenantConfig> = await res.json()
  if (body.error) {
    throw errorWithCode(body.error)
  }
  return body.data
}

export function useTenantConfig() {
  return useQuery<TenantConfig, Error>({
    queryKey: ['tenant', 'config'],
    queryFn: fetchTenantConfig,
    staleTime: 5 * 60 * 1000, // AC-6: 5 minutes — tenant config rarely changes
  })
}
