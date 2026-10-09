import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { errorWithCode } from '@/api/errors'
import { jwtSub } from '@/lib/routeRoles'

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
  is_locked: boolean
  created_at: string
  /** Present on GET /users/me only: the caller's "resource:action" permissions (FR-BB117 AC-16). */
  permissions?: string[]
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

export interface RoleRow {
  id: string
  name: string
  /** Server-computed (FR-BB117 D-1, ISS-229): may the current caller assign this role? Absent on older backends. */
  assignable?: boolean
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

async function apiFetch<T>(url: string, token?: string | null, options?: RequestInit): Promise<T> {
  const authHeader: Record<string, string> = token ? { Authorization: `Bearer ${token}` } : {}
  const res = await fetch(url, {
    ...options,
    credentials: 'include',
    headers: { ...authHeader, ...(options?.headers as Record<string, string>) },
  })
  const body: ApiResponse<T> = await res.json()
  if (body.error) {
    throw errorWithCode(body.error)
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
  const qc = useQueryClient()
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  return useQuery<UsersListResponse, Error>({
    queryKey: ['users', filters],
    queryFn: () => apiFetch<UsersListResponse>(`/api/v1/users${buildQuery(filters)}`, token),
  })
}

export function useUser(id: string) {
  const qc = useQueryClient()
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  return useQuery<User, Error>({
    queryKey: ['users', id],
    queryFn: () => apiFetch<User>(`/api/v1/users/${id}`, token),
    enabled: !!id,
  })
}

export function useMe(options?: { enabled?: boolean }) {
  const qc = useQueryClient()
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  return useQuery<User, Error>({
    queryKey: ['users', 'me'],
    queryFn: () => apiFetch<User>('/api/v1/users/me', token),
    enabled: options?.enabled ?? true,
  })
}

export function useCreateUser() {
  const queryClient = useQueryClient()
  return useMutation<CreateUserResponse, Error, CreateUserRequest>({
    mutationFn: (body) => {
      const token = queryClient.getQueryData<string | null>(['auth', 'accessToken'])
      return apiFetch<CreateUserResponse>('/api/v1/users', token, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] })
    },
  })
}

export function useUpdateUser(id: string) {
  const queryClient = useQueryClient()
  return useMutation<User, Error, UpdateUserRequest>({
    mutationFn: (body) => {
      const token = queryClient.getQueryData<string | null>(['auth', 'accessToken'])
      return apiFetch<User>(`/api/v1/users/${id}`, token, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] })
    },
  })
}

export function useDeactivateUser(id: string) {
  const queryClient = useQueryClient()
  return useMutation<Record<string, never>, Error, void>({
    mutationFn: () => {
      const token = queryClient.getQueryData<string | null>(['auth', 'accessToken'])
      return apiFetch<Record<string, never>>(`/api/v1/users/${id}/deactivate`, token, { method: 'POST' })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] })
    },
  })
}

export function useResetPassword(id: string) {
  const qc = useQueryClient()
  return useMutation<ResetPasswordResponse, Error, void>({
    mutationFn: () => {
      const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
      return apiFetch<ResetPasswordResponse>(`/api/v1/users/${id}/reset-password`, token, { method: 'POST' })
    },
  })
}

// FR-BB115 AC-8: admin early unlock. The user id is the mutation variable so one hook
// instance serves every row of the list.
export function useUnlockUser() {
  const queryClient = useQueryClient()
  return useMutation<User, Error, string>({
    mutationFn: (id) => {
      const token = queryClient.getQueryData<string | null>(['auth', 'accessToken'])
      return apiFetch<User>(`/api/v1/users/${id}/unlock`, token, { method: 'POST' })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] })
    },
  })
}

export const rolesQueryKey = (userId: string | undefined) => ['users', 'roles', userId] as const

export function useRoles() {
  const qc = useQueryClient()
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  // The response carries per-caller `assignable` flags, so the key is scoped to the current
  // user (JWT sub); without an id the query stays disabled rather than use a shared key.
  const userId = jwtSub(token)
  return useQuery<RoleRow[], Error>({
    queryKey: rolesQueryKey(userId),
    queryFn: () => apiFetch<RoleRow[]>('/api/v1/users/roles', token),
    enabled: !!userId,
  })
}

export function useImportUsers() {
  const queryClient = useQueryClient()
  return useMutation<ImportPreview, Error, { file: File; commit: boolean }>({
    mutationFn: ({ file, commit }) => {
      const token = queryClient.getQueryData<string | null>(['auth', 'accessToken'])
      const form = new FormData()
      form.append('file', file)
      const url = `/api/v1/users/import${commit ? '?commit=true' : ''}`
      return apiFetch<ImportPreview>(url, token, { method: 'POST', body: form })
    },
    onSuccess: (_data, variables) => {
      if (variables.commit) {
        queryClient.invalidateQueries({ queryKey: ['users'] })
      }
    },
  })
}
