import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

export interface Tag {
  id: string
  name: string
  created_at: string
  usage_count: number
}

interface ApiResponse<T> {
  data: T
  error: null | { code: string; message: string; details?: Record<string, unknown> }
}

export async function tagsFetch<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(url, options)
  const body: ApiResponse<T> = await res.json()
  if (body.error) {
    const err = new Error(body.error.message) as Error & { code: string; details?: Record<string, unknown> }
    err.code = body.error.code
    err.details = body.error.details
    throw err
  }
  if (!res.ok) {
    throw new Error(`Request failed: ${res.status}`)
  }
  return body.data
}

export function useTags() {
  return useQuery({
    queryKey: ['tags'],
    queryFn: async () => tagsFetch<Tag[]>('/api/v1/tags'),
    staleTime: 300_000,
  })
}

export function useCreateTag() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (name: string) =>
      tagsFetch<Tag>('/api/v1/tags', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name }),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tags'] }),
  })
}

export function useUpdateTag() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) =>
      tagsFetch<Tag>(`/api/v1/tags/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name }),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tags'] }),
  })
}

export function useDeleteTag() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) =>
      tagsFetch<void>(`/api/v1/tags/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tags'] }),
  })
}
