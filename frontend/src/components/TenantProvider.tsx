import { useEffect, type ReactNode } from 'react'
import { useTenantConfig } from '@/api/useTenantConfig'

interface TenantProviderProps {
  children: ReactNode
}

export function TenantProvider({ children }: TenantProviderProps) {
  const { data } = useTenantConfig()

  useEffect(() => {
    if (!data) return
    const root = document.documentElement
    root.style.setProperty('--color-primary', data.primary_color)
    root.style.setProperty('--color-accent', data.accent_color)
  }, [data])

  return <>{children}</>
}
