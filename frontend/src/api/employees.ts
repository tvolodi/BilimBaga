import { useQuery, keepPreviousData, useQueryClient } from '@tanstack/react-query'
import type { QueryClient } from '@tanstack/react-query'
import { downloadFile } from '@/api/download'
import { apiFetch, apiFetchPaginated } from './apiFetch'

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

function fetchPaginated<T>(qc: QueryClient, url: string): Promise<PaginatedResult<T>> {
  return apiFetchPaginated<T, PaginationMeta>(qc, url)
}

// ---- Hooks ------------------------------------------------------------------

export function useEmployeeRecord(userId: string, page: number) {
  const qc = useQueryClient()
  return useQuery({
    queryKey: ['employee-record', userId, page],
    queryFn: () => fetchPaginated<EmployeeRecordData>(qc, `/api/v1/admin/users/${userId}/record?page=${page}&per_page=20`),
    placeholderData: keepPreviousData,
  })
}

export function useEmployeeProgress(userId: string) {
  const qc = useQueryClient()
  return useQuery({
    queryKey: ['employee-progress', userId],
    queryFn: () => apiFetch<EmployeeProgress>(qc, `/api/v1/admin/users/${userId}/progress`),
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
