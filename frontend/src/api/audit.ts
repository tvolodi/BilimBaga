import { useQuery, useQueryClient } from '@tanstack/react-query'
import type { QueryClient } from '@tanstack/react-query'
import { downloadFile } from '@/api/download'
import { apiFetch } from './apiFetch'

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

// The values the audit filter offers for this tenant: the distinct action codes and entity types its log holds, sorted
// (FR-BB55 AC-2, #557). Codes are shown as they are; the backend codes are never renamed.
export interface AuditFilterOptions {
  actions: string[]
  entity_types: string[]
}

// ---- API helpers ------------------------------------------------------------

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
  return useQuery<AuditLogResponse, Error>({
    queryKey: ['audit-log', filters, page],
    queryFn: () => apiFetch<AuditLogResponse>(qc, buildAuditURL(filters, page)),
  })
}

// The filter options change only when a new code is first written, so they are kept for five minutes.
export function useAuditFilterOptions() {
  const qc = useQueryClient()
  return useQuery<AuditFilterOptions, Error>({
    queryKey: ['audit-filter-options'],
    queryFn: () => apiFetch<AuditFilterOptions>(qc, '/api/v1/audit/actions'),
    staleTime: 5 * 60 * 1000,
  })
}

export function exportAuditLog(qc: QueryClient, filters: AuditFilters): Promise<void> {
  return downloadFile(qc, buildExportURL(filters), 'audit-export.csv')
}
