import { useQuery } from '@tanstack/react-query'

export interface VerifyResult {
  valid: boolean
  employee_name?: string
  exam_title?: string
  score_pct?: number
  issued_at?: string
}

interface ApiResponse<T> {
  data: T | null
  error: null | { code: string; message: string }
}

// Plain unauthenticated fetch: the verification page is public and must never
// send an Authorization header or trigger a token refresh.
async function fetchVerify(code: string): Promise<VerifyResult> {
  const res = await fetch(`/api/v1/verify/${encodeURIComponent(code)}`)
  if (!res.ok) {
    throw new Error(`Verification request failed: ${res.status}`)
  }
  const body: ApiResponse<VerifyResult> = await res.json()
  if (body.error || !body.data) {
    throw new Error(body.error?.message ?? 'Empty verification response')
  }
  return body.data
}

export function useVerifyCertificate(code: string) {
  return useQuery<VerifyResult, Error>({
    queryKey: ['verify', code],
    queryFn: () => fetchVerify(code),
    retry: false,
    staleTime: 5 * 60_000,
    enabled: code.length > 0,
  })
}
