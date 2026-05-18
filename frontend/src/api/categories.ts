import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiFetch } from './apiFetch'

// ---- Types ------------------------------------------------------------------

export interface CategoryNode {
  id: string
  name: string
  parent_id: string | null
  track: string | null
  sort_order: number
  created_at: string
  updated_at: string
  children: CategoryNode[]
}

export interface CategoryCreate {
  name: string
  parent_id?: string | null
  track?: string | null
  sort_order: number
}

export interface CategoryUpdate {
  name?: string
  parent_id?: string | null
  track?: string | null
  sort_order?: number
  clear_parent?: boolean
}

// ---- Hooks ------------------------------------------------------------------

export function useCategories() {
  const qc = useQueryClient()
  return useQuery({
    queryKey: ['categories'],
    queryFn: () => apiFetch<CategoryNode[]>(qc, '/api/v1/categories'),
    staleTime: 300_000,
  })
}

export function useCreateCategory() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: CategoryCreate) =>
      apiFetch<CategoryNode>(qc, '/api/v1/categories', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['categories'] }),
  })
}

export function useUpdateCategory() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, body }: { id: string; body: CategoryUpdate }) =>
      apiFetch<CategoryNode>(qc, `/api/v1/categories/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['categories'] }),
  })
}

export function useDeleteCategory() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) =>
      apiFetch<void>(qc, `/api/v1/categories/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['categories'] }),
  })
}

// ---- Helpers ----------------------------------------------------------------

export function flattenCategories(nodes: CategoryNode[]): CategoryNode[] {
  const result: CategoryNode[] = []
  function walk(list: CategoryNode[]) {
    for (const node of list) {
      result.push(node)
      if (node.children?.length) walk(node.children)
    }
  }
  walk(nodes)
  return result
}

export function getDescendantIds(node: CategoryNode): string[] {
  const ids: string[] = []
  function walk(n: CategoryNode) {
    for (const child of n.children ?? []) {
      ids.push(child.id)
      walk(child)
    }
  }
  walk(node)
  return ids
}
