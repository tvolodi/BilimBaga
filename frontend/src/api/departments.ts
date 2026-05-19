import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiFetch } from './apiFetch'

export interface Department {
  id: string
  name: string
  parent_id: string | null
  created_at: string
  updated_at: string
  children: Department[]
}

export function useDepartments() {
  const qc = useQueryClient()
  return useQuery<Department[], Error>({
    queryKey: ['departments'],
    queryFn: () => apiFetch<Department[]>(qc, '/api/v1/departments'),
  })
}

export function useCreateDepartment() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: { name: string; parent_id?: string | null }) =>
      apiFetch<Department>(qc, '/api/v1/departments', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['departments'] }),
  })
}

export function useUpdateDepartment() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) =>
      apiFetch<Department>(qc, `/api/v1/departments/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name }),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['departments'] }),
  })
}

export function useDeleteDepartment() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) =>
      apiFetch<void>(qc, `/api/v1/departments/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['departments'] }),
  })
}
