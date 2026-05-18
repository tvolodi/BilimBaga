import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiFetch } from './apiFetch'

export interface Tag {
  id: string
  name: string
  created_at: string
  usage_count: number
}

export function useTags() {
  const qc = useQueryClient()
  return useQuery({
    queryKey: ['tags'],
    queryFn: () => apiFetch<Tag[]>(qc, '/api/v1/tags'),
    staleTime: 300_000,
  })
}

export function useCreateTag() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (name: string) =>
      apiFetch<Tag>(qc, '/api/v1/tags', {
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
      apiFetch<Tag>(qc, `/api/v1/tags/${id}`, {
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
      apiFetch<void>(qc, `/api/v1/tags/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tags'] }),
  })
}
