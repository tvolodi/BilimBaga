import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

export interface ApiError {
  code: string
  message: string
  details?: Record<string, string>
}

export interface CurrentUser {
  id: string
  full_name: string
  email: string
  role: string
  force_password_change: boolean
}

interface LoginPayload {
  email: string
  password: string
}

interface LoginResponse {
  access_token: string
  token_type: string
  expires_in: number
  user: CurrentUser
}

interface ChangePasswordPayload {
  current_password: string
  new_password: string
}

function decodeJwtPayload(token: string): Record<string, unknown> | null {
  try {
    const payload = token.split('.')[1]
    return JSON.parse(atob(payload.replace(/-/g, '+').replace(/_/g, '/')))
  } catch {
    return null
  }
}

export function useLogin() {
  const qc = useQueryClient()
  return useMutation<LoginResponse, ApiError, LoginPayload>({
    mutationFn: async (payload) => {
      const res = await fetch('/api/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
        credentials: 'include',
      })
      const json = await res.json()
      if (!res.ok) throw json.error as ApiError
      return json.data as LoginResponse
    },
    onSuccess: (data) => {
      qc.setQueryData(['auth', 'currentUser'], data.user)
      qc.setQueryData(['auth', 'accessToken'], data.access_token)
    },
  })
}

export function useChangePassword() {
  const qc = useQueryClient()
  return useMutation<void, ApiError, ChangePasswordPayload>({
    mutationFn: async (payload) => {
      const token = qc.getQueryData<string>(['auth', 'accessToken'])
      const res = await fetch('/api/v1/auth/change-password', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify(payload),
        credentials: 'include',
      })
      const json = await res.json()
      if (!res.ok) throw json.error as ApiError
    },
  })
}

export function useLogout() {
  const qc = useQueryClient()
  return useMutation<void, ApiError, void>({
    mutationFn: async () => {
      await fetch('/api/v1/auth/logout', {
        method: 'POST',
        credentials: 'include',
      })
    },
    onSettled: () => {
      qc.setQueryData(['auth', 'accessToken'], null)
      qc.setQueryData(['auth', 'currentUser'], null)
    },
  })
}

export function useRefreshToken() {
  const qc = useQueryClient()
  return useQuery<string | null, Error>({
    queryKey: ['auth', 'accessToken'],
    queryFn: async () => {
      try {
        const res = await fetch('/api/v1/auth/refresh', {
          method: 'POST',
          credentials: 'include',
        })
        if (!res.ok) return null
        const json = await res.json()
        if (json.error || !json.data) return null
        const accessToken = json.data.access_token as string
        // Reconstruct minimal user info from JWT claims (refresh response has no user object)
        const claims = decodeJwtPayload(accessToken)
        if (claims) {
          qc.setQueryData(['auth', 'currentUser'], {
            id: (claims.sub as string) ?? '',
            email: (claims.email as string) ?? '',
            role: (claims.role as string) ?? '',
            full_name: '',
            force_password_change: false,
          })
        }
        return accessToken
      } catch {
        return null
      }
    },
    staleTime: Infinity,
    retry: false,
  })
}
