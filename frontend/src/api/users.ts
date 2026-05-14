import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

// ---- Types ------------------------------------------------------------------

export interface User {
  id: string
  email: string
  full_name: string
  department_id: string | null
  department_name: string | null
  role_id: string
  role_name: string
  status: 'active' | 'inactive'
  force_password_change: boolean
  created_at: string
}

export interface UserMeta {
  page: number
  per_page: number
  total: number
}

export interface UsersListResponse {
  items: User[]
  meta: UserMeta
}

export interface CreateUserRequest {
  email: string
  full_name: string
  department_id: string | null
  role_id: string
}

export interface CreateUserResponse extends User {
  temporary_password: string
}

export interface UpdateUserRequest {
  full_name: string
  department_id: string | null
  role_id: string
}

export interface ResetPasswordResponse {
  temporary_password: string
}

export interface ImportRowResult {
  row_num: number
  email: string
  full_name: string
  department_name: string
  role_name: string
  error?: string
}

export interface ImportPreview {
  valid: ImportRowResult[]
  errors: ImportRowResult[]
}

export interface UsersFilters {
  department_id?: string
  role_id?: string
  status?: string
  page?: number
  per_page?: number
}

// ---- API helpers ------------------------------------------------------------

interface ApiResponse<T> {
  data: T
  error: null | { code: string; message: string }
}

async function apiFetch<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(url, options)
  const body: ApiResponse<T> = await res.json()
  if (body.error) {
    throw new Error(body.error.message)
  }
  if (!res.ok) {
    throw new Error(`Request failed: ${res.status}`)
  }
  return body.data
}

function buildQuery(filters: UsersFilters): string {
  const params = new URLSearchParams()
  if (filters.department_id) params.set('department_id', filters.department_id)
  if (filters.role_id) params.set('role_id', filters.role_id)
  if (filters.status) params.set('status', filters.status)
  if (filters.page) params.set('page', String(filters.page))
  if (filters.per_page) params.set('per_page', String(filters.per_page))
  const qs = params.toString()
  return qs ? `?${qs}` : ''
}

// ---- Hooks ------------------------------------------------------------------

export function useUsers(filters: UsersFilters = {}) {
  return useQuery<UsersListResponse, Error>({
    queryKey: ['users', filters],
    queryFn: () => apiFetch<UsersListResponse>(`/api/v1/users${buildQuery(filters)}`),
  })
}

export function useUser(id: string) {
  return useQuery<User, Error>({
    queryKey: ['users', id],
    queryFn: () => apiFetch<User>(`/api/v1/users/${id}`),
    enabled: !!id,
  })
}

export function useMe() {
  return useQuery<User, Error>({
    queryKey: ['users', 'me'],
    queryFn: () => apiFetch<User>('/api/v1/users/me'),
  })
}

export function useCreateUser() {
  const queryClient = useQueryClient()
  return useMutation<CreateUserResponse, Error, CreateUserRequest>({
    mutationFn: (body) =>
      apiFetch<CreateUserResponse>('/api/v1/users', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] })
    },
  })
}

export function useUpdateUser(id: string) {
  const queryClient = useQueryClient()
  return useMutation<User, Error, UpdateUserRequest>({
    mutationFn: (body) =>
      apiFetch<User>(`/api/v1/users/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] })
    },
  })
}

export function useDeactivateUser(id: string) {
  const queryClient = useQueryClient()
  return useMutation<Record<string, never>, Error, void>({
    mutationFn: () =>
      apiFetch<Record<string, never>>(`/api/v1/users/${id}/deactivate`, { method: 'POST' }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] })
    },
  })
}

export function useResetPassword(id: string) {
  return useMutation<ResetPasswordResponse, Error, void>({
    mutationFn: () =>
      apiFetch<ResetPasswordResponse>(`/api/v1/users/${id}/reset-password`, { method: 'POST' }),
  })
}

export function useImportUsers() {
  const queryClient = useQueryClient()
  return useMutation<ImportPreview, Error, { file: File; commit: boolean }>({
    mutationFn: ({ file, commit }) => {
      const form = new FormData()
      form.append('file', file)
      const url = `/api/v1/users/import${commit ? '?commit=true' : ''}`
      return apiFetch<ImportPreview>(url, { method: 'POST', body: form })
    },
    onSuccess: (_data, variables) => {
      if (variables.commit) {
        queryClient.invalidateQueries({ queryKey: ['users'] })
      }
    },
  })
}
