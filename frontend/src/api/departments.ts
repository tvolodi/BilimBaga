import { useQuery } from '@tanstack/react-query'

// ---- Types ------------------------------------------------------------------

export interface Department {
  id: string
  name: string
  parent_id: string | null
  children?: Department[]
}

// ---- API helper -------------------------------------------------------------

interface ApiResponse<T> {
  data: T
  error: null | { code: string; message: string }
}

async function apiFetch<T>(url: string): Promise<T> {
  const res = await fetch(url)
  const body: ApiResponse<T> = await res.json()
  if (body.error) {
    throw new Error(body.error.message)
  }
  if (!res.ok) {
    throw new Error(`Request failed: ${res.status}`)
  }
  return body.data
}

// ---- Hooks ------------------------------------------------------------------

export function useDepartments() {
  return useQuery<Department[], Error>({
    queryKey: ['departments'],
    queryFn: () => apiFetch<Department[]>('/api/v1/departments'),
  })
}
