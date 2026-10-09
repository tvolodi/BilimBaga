import { useMutation, useQuery, useQueryClient, type QueryClient } from '@tanstack/react-query'
import { apiFetch } from './apiFetch'

// FR-BB117 role management API.

export interface AdminRole {
  id: string
  name: string
  description: string
  is_system: boolean
  user_count: number
  /** "resource:action" strings. */
  permissions: string[]
  created_at: string
}

export interface CataloguePermission {
  id: string
  resource: string
  action: string
}

export interface CreateRoleInput {
  name: string
  description: string
  /** Permission ids from the catalogue. */
  permissions: string[]
}

export interface UpdateRoleInput {
  id: string
  description: string
  permissions: string[]
}

/** Permissions the backend refuses to grant to a custom role (FR-BB117 AC-6). */
export const NON_ASSIGNABLE_PERMISSIONS: readonly string[] = ['roles:read', 'roles:manage', 'tenant:manage']

/** Role name rule enforced by the backend (FR-BB117 AC-3). */
export const ROLE_NAME_PATTERN = /^[a-z][a-z0-9_]{2,31}$/

const JSON_HEADERS = { 'Content-Type': 'application/json' }

function invalidateRoleCaches(qc: QueryClient) {
  qc.invalidateQueries({ queryKey: ['roles'] })
  // Role selects in the user create/edit drawers (api/users.ts useRoles).
  qc.invalidateQueries({ queryKey: ['users', 'roles'] })
}

export function useRolesAdmin() {
  const qc = useQueryClient()
  return useQuery({
    queryKey: ['roles'],
    queryFn: () => apiFetch<AdminRole[]>(qc, '/api/v1/roles'),
  })
}

export function usePermissionsCatalogue() {
  const qc = useQueryClient()
  return useQuery({
    queryKey: ['roles', 'permissions'],
    queryFn: () => apiFetch<CataloguePermission[]>(qc, '/api/v1/roles/permissions'),
    staleTime: 300_000,
  })
}

export function useCreateRole() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: CreateRoleInput) =>
      apiFetch<AdminRole>(qc, '/api/v1/roles', {
        method: 'POST',
        headers: JSON_HEADERS,
        body: JSON.stringify(body),
      }),
    onSuccess: () => invalidateRoleCaches(qc),
  })
}

export function useUpdateRole() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...body }: UpdateRoleInput) =>
      apiFetch<AdminRole>(qc, `/api/v1/roles/${id}`, {
        method: 'PUT',
        headers: JSON_HEADERS,
        body: JSON.stringify(body),
      }),
    onSuccess: () => invalidateRoleCaches(qc),
  })
}

export function useDeleteRole() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => apiFetch<void>(qc, `/api/v1/roles/${id}`, { method: 'DELETE' }),
    onSuccess: () => invalidateRoleCaches(qc),
  })
}

/** Codes the roles API can return that have a dedicated translated message (`roles.errors.<CODE>`). */
const KNOWN_ERROR_CODES = [
  'ROLE_NAME_TAKEN',
  'ROLE_SYSTEM_IMMUTABLE',
  'ROLE_IN_USE',
  'VALIDATION_ERROR',
  'NOT_FOUND',
  'FORBIDDEN',
] as const

/** Maps a server error to an i18n key + interpolation; unknown codes fall back to a generic key. */
export function roleErrorKey(err: unknown): { key: string; values?: Record<string, unknown> } {
  const e = err as { code?: string; details?: { count?: unknown } } | null
  const code = e?.code
  if (code && (KNOWN_ERROR_CODES as readonly string[]).includes(code)) {
    if (code === 'ROLE_IN_USE') {
      const count = Number(e?.details?.count)
      return { key: 'roles.errors.ROLE_IN_USE', values: { count: Number.isFinite(count) ? count : 0 } }
    }
    return { key: `roles.errors.${code}` }
  }
  return { key: 'roles.errors.generic' }
}
