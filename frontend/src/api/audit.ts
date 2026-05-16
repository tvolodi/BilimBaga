import { useQuery, useQueryClient } from '@tanstack/react-query'

// ---- Types ------------------------------------------------------------------

export interface AuditEntry {
  id: string
  created_at: string
  actor_id: string | null
  actor_name: string | null
  action: string
  entity_type: string | null
  entity_id: string | null
  ip_address: string | null
  metadata: Record<string, unknown>
}

export interface AuditFilters {
  from?: string
  to?: string
  actor?: string
  actions?: string[]
  entityType?: string
}

export interface AuditMeta {
  page: number
  per_page: number
  total: number
}

export interface AuditLogResponse {
  items: AuditEntry[]
  meta: AuditMeta
}

export const AUDIT_ACTIONS = [
  'user.login',
  'user.logout',
  'user.create',
  'user.update',
  'user.deactivate',
  'exam.create',
  'exam.update',
  'exam.publish',
  'exam.archive',
  'session.start',
  'session.submit',
  'session.expire',
  'session.pending_manual_grade',
  'answer.grade',
  'certificate.issue',
  'question.create',
  'question.update',
  'question.delete',
  'tenant.update',
] as const

// ---- API helpers ------------------------------------------------------------

interface ApiResponse<T> {
  data: T
  error: null | { code: string; message: string }
}

async function apiFetch<T>(url: string, token?: string | null): Promise<T> {
  const res = await fetch(url, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    credentials: 'include',
  })
  const body: ApiResponse<T> = await res.json()
  if (body.error) {
    throw new Error(body.error.message)
  }
  if (!res.ok) {
    throw new Error(`Request failed: ${res.status}`)
  }
  return body.data
}

function buildAuditURL(filters: AuditFilters, page: number): string {
  const p = new URLSearchParams()
  if (filters.from) p.set('from', filters.from)
  if (filters.to) p.set('to', filters.to)
  if (filters.actor) p.set('actor', filters.actor)
  if (filters.actions?.length) filters.actions.forEach((a) => p.append('action', a))
  if (filters.entityType) p.set('entity_type', filters.entityType)
  p.set('page', String(page))
  p.set('per_page', '50')
  return `/api/v1/audit?${p.toString()}`
}

function buildExportURL(filters: AuditFilters): string {
  const p = new URLSearchParams()
  if (filters.from) p.set('from', filters.from)
  if (filters.to) p.set('to', filters.to)
  if (filters.actor) p.set('actor', filters.actor)
  if (filters.actions?.length) filters.actions.forEach((a) => p.append('action', a))
  if (filters.entityType) p.set('entity_type', filters.entityType)
  return `/api/v1/audit/export?${p.toString()}`
}

// ---- Hooks ------------------------------------------------------------------

export function useAuditLog(filters: AuditFilters, page: number) {
  const qc = useQueryClient()
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  return useQuery<AuditLogResponse, Error>({
    queryKey: ['audit-log', filters, page],
    queryFn: () => apiFetch<AuditLogResponse>(buildAuditURL(filters, page), token),
  })
}

export async function exportAuditLog(
  filters: AuditFilters,
  token: string | null | undefined,
): Promise<void> {
  const res = await fetch(buildExportURL(filters), {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    credentials: 'include',
  })
  if (!res.ok) {
    throw new Error(`Export failed: ${res.status}`)
  }
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  // Extract filename from Content-Disposition header or use a default.
  const disposition = res.headers.get('Content-Disposition') ?? ''
  const match = /filename="([^"]+)"/.exec(disposition)
  a.download = match ? match[1] : 'audit-export.csv'
  a.href = url
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}
