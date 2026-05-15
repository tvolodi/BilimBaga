import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

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

// ---- API helpers ------------------------------------------------------------

interface ApiResponse<T> {
  data: T
  error: null | { code: string; message: string; details?: Record<string, unknown> }
}

export interface ApiError {
  code: string
  message: string
  details?: Record<string, unknown>
}

export async function categoriesFetch<T>(url: string, options?: RequestInit): Promise<T> {
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

// ---- Hooks ------------------------------------------------------------------

export function useCategories() {
  return useQuery({
    queryKey: ['categories'],
    queryFn: async () => categoriesFetch<CategoryNode[]>('/api/v1/categories'),
    staleTime: 300_000,
  })
}

export function useCreateCategory() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: CategoryCreate) =>
      categoriesFetch<CategoryNode>('/api/v1/categories', {
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
      categoriesFetch<CategoryNode>(`/api/v1/categories/${id}`, {
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
      categoriesFetch<void>(`/api/v1/categories/${id}`, { method: 'DELETE' }),
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
