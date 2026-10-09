import { useMutation } from '@tanstack/react-query'
import type { ApiError } from '@/api/auth'

// FR-BB115: public account-recovery calls. These deliberately use plain fetch with no
// Authorization header and no credentials — the user is not signed in.

interface MessageResponse {
  message: string
}

async function postPublic<T>(url: string, body: unknown): Promise<T> {
  const res = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  let json: { data: T | null; error: ApiError | null } | null = null
  try {
    json = await res.json()
  } catch {
    json = null
  }
  if (!res.ok || json?.error) {
    throw (json?.error ?? { code: 'ERR_HTTP', message: `Request failed: ${res.status}` }) as ApiError
  }
  return json?.data as T
}

export function useForgotPassword() {
  return useMutation<MessageResponse, ApiError, { email: string }>({
    mutationFn: (payload) => postPublic<MessageResponse>('/api/v1/auth/forgot-password', payload),
  })
}

export function useCompletePasswordReset() {
  return useMutation<MessageResponse, ApiError, { token: string; new_password: string }>({
    mutationFn: (payload) => postPublic<MessageResponse>('/api/v1/auth/reset-password', payload),
  })
}
