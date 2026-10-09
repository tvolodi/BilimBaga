import { useQuery, keepPreviousData, useQueryClient } from '@tanstack/react-query'
import type { QueryClient } from '@tanstack/react-query'
import { downloadFile } from '@/api/download'
import { errorWithCode } from '@/api/errors'

// ---- Types ------------------------------------------------------------------

export interface EmployeeRecordData {
  sessions: SessionRecord[]
}

export interface PaginationMeta {
  page: number
  per_page: number
  total: number
}

export interface PaginatedResult<T> {
  data: T
  meta: PaginationMeta
}

export interface SessionRecord {
  session_id: string
  exam_id: string
  exam_title: string
  started_at: string
  submitted_at: string | null
  score_pct: number | null
  passed: boolean | null
  time_taken_seconds: number | null
  status: string
  certificate_id: string | null
  exam_category_track: string | null
}

export interface EmployeeProgress {
  user_id: string
  full_name: string
  tracks: TrackSummary[]
}

export interface TrackSummary {
  track: 'security' | 'safety' | 'loyalty'
  questions_answered: number
  last_activity: string | null
  required_exams: RequiredExam[]
}

export interface RequiredExam {
  exam_id: string
  title: string
  passed: boolean | null
  attempts: number
}

// ---- API helpers ------------------------------------------------------------

interface ApiPaginatedResponse<T> {
  data: T
  meta: PaginationMeta
  error: null | { code: string; message: string }
}

interface ApiResponse<T> {
  data: T
  error: null | { code: string; message: string }
}

async function fetchPaginated<T>(url: string, token?: string | null): Promise<PaginatedResult<T>> {
  const res = await fetch(url, {
    credentials: 'include',
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  })
  const body: ApiPaginatedResponse<T> = await res.json()
  if (body.error) throw errorWithCode(body.error)
  if (!res.ok) throw new Error(`Request failed: ${res.status}`)
  return { data: body.data, meta: body.meta }
}

async function apiFetch<T>(url: string, token?: string | null): Promise<T> {
  const res = await fetch(url, {
    credentials: 'include',
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  })
  const body: ApiResponse<T> = await res.json()
  if (body.error) throw errorWithCode(body.error)
  if (!res.ok) throw new Error(`Request failed: ${res.status}`)
  return body.data
}

// ---- Hooks ------------------------------------------------------------------

export function useEmployeeRecord(userId: string, page: number) {
  const qc = useQueryClient()
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  return useQuery({
    queryKey: ['employee-record', userId, page],
    queryFn: () =>
      fetchPaginated<EmployeeRecordData>(
        `/api/v1/admin/users/${userId}/record?page=${page}&per_page=20`,
        token,
      ),
    placeholderData: keepPreviousData,
  })
}

export function useEmployeeProgress(userId: string) {
  const qc = useQueryClient()
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  return useQuery({
    queryKey: ['employee-progress', userId],
    queryFn: () => apiFetch<EmployeeProgress>(`/api/v1/admin/users/${userId}/progress`, token),
    staleTime: 5 * 60 * 1000,
  })
}

// ---- Employee record export -------------------------------------------------

export function exportEmployeeRecord(qc: QueryClient, userId: string): Promise<void> {
  return downloadFile(
    qc,
    `/api/v1/admin/users/${userId}/record/export`,
    `employee-${userId}-record.csv`,
  )
}

// ---- Certificate download ---------------------------------------------------

export function downloadAdminCertificate(
  qc: QueryClient,
  sessionId: string,
  verificationCode: string,
): Promise<void> {
  return downloadFile(
    qc,
    `/api/v1/admin/sessions/${sessionId}/certificate`,
    `certificate-${verificationCode}.pdf`,
  )
}
