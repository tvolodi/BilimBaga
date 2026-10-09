/**
 * Employee fixture seeding for E2E tests.
 *
 * Called once from global-setup.ts after admin login.
 * Creates (idempotently):
 *  - employee@bilimbaga.local user
 *  - "E2E Mixed Exam"  (one question per type)
 *  - "E2E ShortText Exam"  (one short_text question, pre-submitted session)
 *  - saves employee JWT to .auth/employee.json and .auth/employee-token.txt
 */

import { chromium } from '@playwright/test'
import { requireTarget } from '../../../scripts/lib/target-guard'
import path from 'path'
import { fileURLToPath } from 'url'
import fs from 'fs'

const APP_URL = requireTarget('E2E_BASE_URL', process.env.E2E_BASE_URL || 'http://localhost:5173', process.env)

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const AUTH_DIR = path.join(__dirname, '..', '..', '.auth')
export const EMPLOYEE_STORAGE_STATE = path.join(AUTH_DIR, 'employee.json')
const EMPLOYEE_TOKEN_PATH = path.join(AUTH_DIR, 'employee-token.txt')
const EMPLOYEE_ID_PATH = path.join(AUTH_DIR, 'employee-id.txt')

const BASE = requireTarget('E2E_API_URL', process.env.E2E_API_URL || `http://localhost:${process.env.BB_API_PORT || 8080}`, process.env)

// ---------------------------------------------------------------------------
// Low-level fetch helpers (Node 18+ global fetch)
// ---------------------------------------------------------------------------

async function apiPost<T>(
  url: string,
  body: unknown,
  token?: string,
): Promise<{ ok: boolean; status: number; data: T | null; error: string | null }> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  if (token) headers['Authorization'] = `Bearer ${token}`
  const res = await fetch(url, { method: 'POST', headers, body: JSON.stringify(body) })
  const json = (await res.json()) as { data: T; error: { code: string; message: string } | null }
  return { ok: res.ok, status: res.status, data: json.data ?? null, error: json.error?.code ?? null }
}

async function apiGet<T>(
  url: string,
  token: string,
): Promise<{ ok: boolean; data: T | null }> {
  const res = await fetch(url, {
    headers: { Authorization: `Bearer ${token}` },
  })
  const json = (await res.json()) as { data: T; error: unknown }
  return { ok: res.ok, data: json.data ?? null }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

interface LoginData {
  access_token: string
}

async function login(email: string, pass: string): Promise<string | null> {
  const result = await apiPost<LoginData>(`${BASE}/api/v1/auth/login`, { email, password: pass })
  if (result.ok && result.data?.access_token) return result.data.access_token
  return null
}

interface UserData {
  id: string
  email: string
  temporary_password?: string
}

interface RoleData {
  id: string
  name: string
}

async function getEmployeeRoleId(adminToken: string): Promise<string> {
  const res = await apiGet<RoleData[]>(`${BASE}/api/v1/users/roles`, adminToken)
  if (!res.ok || !res.data) throw new Error('[seed] Failed to fetch roles')
  const role = res.data.find((r) => r.name === 'employee')
  if (!role) throw new Error('[seed] employee role not found')
  return role.id
}

interface ResetPasswordData {
  temporary_password: string
}

async function setEmployeeKnownPassword(adminToken: string, employeeId: string): Promise<void> {
  const reset = await apiPost<ResetPasswordData>(
    `${BASE}/api/v1/users/${employeeId}/reset-password`,
    {},
    adminToken,
  )
  if (!reset.ok || !reset.data?.temporary_password) {
    throw new Error('[seed] Failed to reset employee password')
  }
  const tempPassword = reset.data.temporary_password
  const empToken = await login('employee@bilimbaga.local', tempPassword)
  if (!empToken) throw new Error('[seed] Could not login with reset temp password')
  const changed = await apiPost<unknown>(
    `${BASE}/api/v1/auth/change-password`,
    { current_password: tempPassword, new_password: 'Employee1234!' },
    empToken,
  )
  if (!changed.ok) throw new Error(`[seed] Failed to change employee password: ${changed.error}`)
}

// Returns the password to use for login (always 'Employee1234!' after setup).
async function createEmployee(adminToken: string): Promise<string> {
  // Try login first (idempotency — user already exists with known password)
  const token = await login('employee@bilimbaga.local', 'Employee1234!')
  if (token) {
    console.log('[seed] Employee already exists with known password — skipping creation')
    return 'Employee1234!'
  }
  console.log('[seed] Creating employee user…')
  const roleId = await getEmployeeRoleId(adminToken)
  const result = await apiPost<UserData>(
    `${BASE}/api/v1/users`,
    {
      email: 'employee@bilimbaga.local',
      full_name: 'E2E Employee',
      role_id: roleId,
    },
    adminToken,
  )
  if (!result.ok) {
    if (result.error === 'DUPLICATE_EMAIL') {
      // User exists but password is unknown — reset via admin then set to known value.
      console.log('[seed] Employee exists with unknown password — resetting via admin…')
      const employeeId = await findUserByEmail(adminToken, 'employee@bilimbaga.local')
      if (!employeeId) throw new Error('[seed] Could not find existing employee user')
      await setEmployeeKnownPassword(adminToken, employeeId)
      return 'Employee1234!'
    }
    throw new Error(`Failed to create employee user: ${result.error}`)
  }
  const tempPassword = result.data?.temporary_password
  if (!tempPassword) throw new Error('[seed] No temporary_password in create response')
  // Change password to the known test password so tests are repeatable.
  const empToken = await login('employee@bilimbaga.local', tempPassword)
  if (!empToken) throw new Error('[seed] Could not login with temporary password')
  const changed = await apiPost<unknown>(
    `${BASE}/api/v1/auth/change-password`,
    { current_password: tempPassword, new_password: 'Employee1234!' },
    empToken,
  )
  if (!changed.ok) throw new Error(`[seed] Failed to change employee password: ${changed.error}`)
  return 'Employee1234!'
}

interface ExamData {
  id: string
  title: string
}

interface ExamListResponse {
  items: ExamData[]
  meta: { total: number }
}

async function findExam(adminToken: string, title: string): Promise<string | null> {
  let page = 1
  while (true) {
    const res = await apiGet<ExamListResponse>(`${BASE}/api/v1/exams?per_page=100&page=${page}`, adminToken)
    if (!res.ok || !res.data) return null
    const found = res.data.items.find((e) => e.title === title)
    if (found) return found.id
    const { total } = res.data.meta
    if (page * 100 >= total) return null
    page++
  }
}

interface QuestionData {
  id: string
}

interface CategoryItem {
  id: string
  name: string
}

let cachedCategoryId: string | null = null

async function getFirstCategoryId(adminToken: string): Promise<string> {
  if (cachedCategoryId) return cachedCategoryId
  const res = await apiGet<CategoryItem[]>(`${BASE}/api/v1/categories`, adminToken)
  if (!res.ok || !res.data || res.data.length === 0) {
    throw new Error('[seed] No categories found — cannot create questions')
  }
  cachedCategoryId = res.data[0].id
  return cachedCategoryId
}

async function createQuestion(
  adminToken: string,
  type: string,
  stem: string,
  options?: Array<{ text: string; is_correct: boolean }>,
): Promise<string> {
  const categoryId = await getFirstCategoryId(adminToken)
  const translations: Record<string, { stem: string; explanation: string }> = {
    en: { stem, explanation: '' },
  }

  let answer_options: Array<{
    sort_order: number
    is_correct: boolean
    likert_weight: number | null
    likert_polarity: string | null
    translations: Record<string, { text: string }>
  }> = []

  if (type === 'single' || type === 'multiple' || type === 'truefalse') {
    const opts = options ?? [
      { text: 'Option A', is_correct: true },
      { text: 'Option B', is_correct: false },
    ]
    answer_options = opts.map((o, i) => ({
      sort_order: i + 1,
      is_correct: o.is_correct,
      likert_weight: null,
      likert_polarity: null,
      translations: { en: { text: o.text } },
    }))
  } else if (type === 'likert') {
    answer_options = [
      { sort_order: 1, is_correct: false, likert_weight: 1, likert_polarity: 'positive', translations: { en: { text: 'Strongly Agree' } } },
      { sort_order: 2, is_correct: false, likert_weight: 2, likert_polarity: 'positive', translations: { en: { text: 'Agree' } } },
      { sort_order: 3, is_correct: false, likert_weight: 3, likert_polarity: 'positive', translations: { en: { text: 'Neutral' } } },
      { sort_order: 4, is_correct: false, likert_weight: 4, likert_polarity: 'negative', translations: { en: { text: 'Disagree' } } },
      { sort_order: 5, is_correct: false, likert_weight: 5, likert_polarity: 'negative', translations: { en: { text: 'Strongly Disagree' } } },
    ]
  }

  const result = await apiPost<QuestionData>(
    `${BASE}/api/v1/questions`,
    {
      type,
      category_id: categoryId,
      difficulty: 'easy',
      default_locale: 'en',
      translations,
      answer_options,
    },
    adminToken,
  )
  if (!result.ok || !result.data?.id) {
    throw new Error(`Failed to create ${type} question: ${result.error}`)
  }
  const questionId = result.data.id
  await activateQuestion(adminToken, questionId)
  return questionId
}

async function activateQuestion(adminToken: string, questionId: string): Promise<void> {
  // draft → review
  const toReview = await apiPost<unknown>(
    `${BASE}/api/v1/questions/${questionId}/status`,
    { status: 'review' },
    adminToken,
  )
  if (!toReview.ok) {
    throw new Error(`Failed to transition question ${questionId} to review: ${toReview.error}`)
  }
  // review → active
  const toActive = await apiPost<unknown>(
    `${BASE}/api/v1/questions/${questionId}/status`,
    { status: 'active' },
    adminToken,
  )
  if (!toActive.ok) {
    throw new Error(`Failed to activate question ${questionId}: ${toActive.error}`)
  }
}

interface ExamDetailData {
  id: string
  rules: Array<{ id: string }>
}

async function getExamRuleCount(adminToken: string, examId: string): Promise<number> {
  const res = await apiGet<ExamDetailData>(`${BASE}/api/v1/exams/${examId}`, adminToken)
  if (!res.ok || !res.data) return 0
  return res.data.rules?.length ?? 0
}

interface RuleData {
  id: string
}

async function addRule(adminToken: string, examId: string, questionId: string, sortOrder: number): Promise<void> {
  const result = await apiPost<RuleData>(
    `${BASE}/api/v1/exams/${examId}/rules`,
    {
      mode: 'manual',
      count: 1,
      sort_order: sortOrder,
      questions: [{ question_id: questionId, sort_order: 1 }],
    },
    adminToken,
  )
  if (!result.ok) {
    throw new Error(`Failed to add rule for question ${questionId}: ${result.error}`)
  }
}

interface AssignmentData {
  id: string
}

async function isAlreadyAssigned(adminToken: string, examId: string, userId: string): Promise<boolean> {
  const res = await apiGet<Array<{ assignee_id: string | null }>>(
    `${BASE}/api/v1/exams/${examId}/assignments`,
    adminToken,
  )
  if (!res.ok || !res.data) return false
  return res.data.some((a) => a.assignee_id === userId)
}

async function assignExam(adminToken: string, examId: string, userId: string): Promise<void> {
  if (await isAlreadyAssigned(adminToken, examId, userId)) {
    console.log('[seed] Exam already assigned to employee — skipping')
    return
  }
  const result = await apiPost<AssignmentData>(
    `${BASE}/api/v1/exams/${examId}/assign`,
    { assignee_type: 'user', assignee_id: userId, deadline: null },
    adminToken,
  )
  if (!result.ok && result.error !== 'ASSIGNMENT_ALREADY_EXISTS') {
    throw new Error(`Failed to assign exam: ${result.error}`)
  }
}

async function publishExam(adminToken: string, examId: string): Promise<void> {
  const result = await apiPost<ExamData>(
    `${BASE}/api/v1/exams/${examId}/publish`,
    {},
    adminToken,
  )
  if (!result.ok && result.error !== 'EXAM_NOT_DRAFT') {
    throw new Error(`Failed to publish exam: ${result.error}`)
  }
}

interface ExamCreated {
  id: string
}

interface CreateExamOptions {
  maxAttempts?: number
  certificateEnabled?: boolean
  passingScorePct?: number
}

async function createExam(
  adminToken: string,
  title: string,
  onTabSwitch: string,
  opts: CreateExamOptions = {},
): Promise<string> {
  const result = await apiPost<ExamCreated>(
    `${BASE}/api/v1/exams`,
    {
      title,
      description: 'E2E test exam — created by seed fixture',
      time_limit_minutes: 60,
      passing_score_pct: opts.passingScorePct ?? 70,
      max_attempts: opts.maxAttempts ?? 99,
      available_from: null,
      available_until: null,
      shuffle_questions: false,
      shuffle_options: false,
      show_answers: 'after_completion',
      on_tab_switch: onTabSwitch,
      certificate_enabled: opts.certificateEnabled ?? false,
    },
    adminToken,
  )
  if (!result.ok || !result.data?.id) {
    throw new Error(`Failed to create exam "${title}": ${result.error}`)
  }
  return result.data.id
}


interface UserListResponse {
  items: Array<{ id: string; email: string }>
}

async function findUserByEmail(adminToken: string, email: string): Promise<string | null> {
  let page = 1
  while (true) {
    const res = await apiGet<UserListResponse>(
      `${BASE}/api/v1/users?per_page=100&page=${page}`,
      adminToken,
    )
    if (!res.ok || !res.data) return null
    const found = res.data.items.find((u) => u.email === email)
    if (found) return found.id
    const { total } = res.data.meta
    if (page * 100 >= total) return null
    page++
  }
}

interface SessionData {
  session_id: string
}

interface SessionStateData {
  session_id: string
  questions: unknown[]
}

async function getSession(employeeToken: string, sessionId: string): Promise<SessionStateData | null> {
  const res = await apiGet<SessionStateData>(`${BASE}/api/v1/portal/sessions/${sessionId}`, employeeToken)
  if (!res.ok) return null
  return res.data
}

async function getOpenSessionId(employeeToken: string, examId: string): Promise<string | null> {
  const res = await apiGet<{ open_session_id: string | null }[]>(`${BASE}/api/v1/portal/exams`, employeeToken)
  if (!res.ok || !res.data) return null
  const exams = res.data as Array<{ id: string; open_session_id: string | null }>
  const exam = exams.find((e) => e.id === examId)
  return exam?.open_session_id ?? null
}

async function submitSession(employeeToken: string, sessionId: string): Promise<void> {
  const result = await apiPost<unknown>(
    `${BASE}/api/v1/portal/sessions/${sessionId}/submit`,
    {},
    employeeToken,
  )
  if (!result.ok) {
    throw new Error(`Failed to submit session: ${result.error}`)
  }
}

async function startSession(employeeToken: string, examId: string): Promise<string | null> {
  const result = await apiPost<SessionData>(
    `${BASE}/api/v1/portal/exams/${examId}/sessions`,
    {},
    employeeToken,
  )
  if (!result.ok) {
    if (result.error === 'sessionAlreadyOpen' || result.error === 'SESSION_ALREADY_OPEN') {
      // Check if the open session has questions — if not, submit it and create fresh
      const openSessionId = await getOpenSessionId(employeeToken, examId)
      if (openSessionId) {
        const state = await getSession(employeeToken, openSessionId)
        if (state && state.questions.length === 0) {
          console.log('[seed] Open session has 0 questions (stale from before rules were added) — submitting and creating fresh session')
          await submitSession(employeeToken, openSessionId).catch(() => {
            console.log('[seed] Could not submit stale session (may already be expired/submitted)')
          })
          // Try starting a fresh session
          const retry = await apiPost<SessionData>(
            `${BASE}/api/v1/portal/exams/${examId}/sessions`,
            {},
            employeeToken,
          )
          if (retry.ok && retry.data?.session_id) {
            console.log('[seed] Fresh session created after submitting stale one')
            return retry.data.session_id
          }
        }
      }
      console.log('[seed] Session already open for this exam — skipping session creation')
      return openSessionId
    }
    if (result.error === 'ATTEMPTS_EXHAUSTED' || result.error === 'MAX_ATTEMPTS_REACHED') {
      console.log('[seed] Max attempts exhausted for this exam — skipping session creation')
      return null
    }
    if (result.error === 'INSUFFICIENT_QUESTIONS') {
      // ISS-132: the server now refuses to start an exam that resolves to zero questions (e.g. the
      // seeded exam's question went draft after the editor specs). Do not kill global-setup.
      console.warn(`[seed] WARNING: exam ${examId} cannot start (INSUFFICIENT_QUESTIONS) — skipping session creation`)
      return null
    }
    throw new Error(`Failed to start session: ${result.error}`)
  }
  return result.data?.session_id ?? null
}

// ---------------------------------------------------------------------------
// Per-test helpers — create & clean up isolated data for destructive tests
// ---------------------------------------------------------------------------

export interface TestUser {
  id: string
  email: string
  password: string
  token: string
}

export async function createTestUser(
  adminToken: string,
  emailPrefix: string,
  _roleName = 'employee',
): Promise<TestUser> {
  const roleId = await getEmployeeRoleId(adminToken)
  const email = `${emailPrefix}-${Date.now()}@e2e-test.local`
  const res = await apiPost<UserData & { temporary_password: string }>(
    `${BASE}/api/v1/users`,
    { email, full_name: `Test ${emailPrefix}`, role_id: roleId },
    adminToken,
  )
  if (!res.ok || !res.data?.id) throw new Error(`createTestUser failed: ${res.error}`)
  // Return the temp password directly — avoids login calls that can hit the auth rate limit.
  // Tests that need to authenticate as this user should call login() themselves.
  return { id: res.data.id, email, password: res.data.temporary_password ?? '', token: '' }
}

export async function deleteTestUser(adminToken: string, userId: string): Promise<void> {
  await fetch(`${BASE}/api/v1/users/${userId}`, {
    method: 'DELETE',
    headers: { Authorization: `Bearer ${adminToken}` },
  })
}

export interface TestQuestion {
  id: string
  stem: string
}

export async function createTestQuestion(
  adminToken: string,
  stem: string,
  type = 'single',
  activate = false,
): Promise<TestQuestion> {
  const categoryId = await getFirstCategoryId(adminToken)
  const translations: Record<string, { stem: string; explanation: string }> = {
    en: { stem, explanation: '' },
  }
  let answer_options: Array<{
    sort_order: number
    is_correct: boolean
    likert_weight: number | null
    likert_polarity: string | null
    translations: Record<string, { text: string }>
  }> = [
    { sort_order: 1, is_correct: true, likert_weight: null, likert_polarity: null, translations: { en: { text: 'Option A' } } },
    { sort_order: 2, is_correct: false, likert_weight: null, likert_polarity: null, translations: { en: { text: 'Option B' } } },
  ]
  const result = await apiPost<QuestionData>(
    `${BASE}/api/v1/questions`,
    { type, category_id: categoryId, difficulty: 'easy', default_locale: 'en', translations, answer_options },
    adminToken,
  )
  if (!result.ok || !result.data?.id) throw new Error(`createTestQuestion failed: ${result.error}`)
  const id = result.data.id
  if (activate) await activateQuestion(adminToken, id)
  return { id, stem }
}

export async function deleteTestQuestion(adminToken: string, questionId: string): Promise<void> {
  await fetch(`${BASE}/api/v1/questions/${questionId}`, {
    method: 'DELETE',
    headers: { Authorization: `Bearer ${adminToken}` },
  })
}

export interface TestExam {
  id: string
  title: string
}

export interface TestExamOptions {
  /** Defaults to 99 (the seeded exams' value). */
  maxAttempts?: number
  /** Defaults to false. A certificate needs a passed, submitted session on such an exam. */
  certificateEnabled?: boolean
  passingScorePct?: number
  onTabSwitch?: 'log' | 'warn' | 'submit'
  /** Assign the (published) exam to this user. Requires `questionId` (only then is it published). */
  assignToUserId?: string
}

export async function createTestExam(
  adminToken: string,
  title: string,
  questionId?: string,
  opts: TestExamOptions = {},
): Promise<TestExam> {
  const examId = await createExam(adminToken, title, opts.onTabSwitch ?? 'log', opts)
  if (questionId) {
    await addRule(adminToken, examId, questionId, 1)
    await publishExam(adminToken, examId)
    if (opts.assignToUserId) await assignExam(adminToken, examId, opts.assignToUserId)
  }
  return { id: examId, title }
}

// ---------------------------------------------------------------------------
// Seed-employee session helpers (strict: throw instead of returning null)
// ---------------------------------------------------------------------------

const SEED_EMPLOYEE_EMAIL = 'employee@bilimbaga.local'
const SEED_EMPLOYEE_PASSWORD = 'Employee1234!'
const EMPLOYEE_TOKEN_TTL_MS = 10 * 60_000 // access JWT lives 15 min; re-login before that
let employeeTokenCache: { token: string; at: number } | null = null

/** API access token of the seeded employee (cached; avoids hitting the auth rate limit). */
export async function getEmployeeApiToken(): Promise<string> {
  if (employeeTokenCache && Date.now() - employeeTokenCache.at < EMPLOYEE_TOKEN_TTL_MS) {
    return employeeTokenCache.token
  }
  const token = await login(SEED_EMPLOYEE_EMAIL, SEED_EMPLOYEE_PASSWORD)
  if (!token) throw new Error(`getEmployeeApiToken: login as ${SEED_EMPLOYEE_EMAIL} failed - did global setup run?`)
  employeeTokenCache = { token, at: Date.now() }
  return token
}

interface SessionQuestions {
  session_id: string
  questions: Array<{ id: string; options: Array<{ id: string; text: string }> }>
}

/**
 * Open an exam session for the seeded employee, or return the already-open one. Throws (never
 * returns null) when no session can be had, so specs fail loudly instead of silently skipping.
 */
export async function startEmployeeSession(examId: string): Promise<string> {
  const token = await getEmployeeApiToken()
  const started = await apiPost<SessionQuestions>(`${BASE}/api/v1/portal/exams/${examId}/sessions`, {}, token)
  if (started.ok && started.data?.session_id) return started.data.session_id
  if (started.error === 'SESSION_ALREADY_OPEN') {
    const open = await getOpenSessionId(token, examId)
    if (open) return open
  }
  throw new Error(`startEmployeeSession: cannot open a session for exam ${examId}: ${started.error ?? started.status}`)
}

/**
 * Give the seeded employee a submitted, passed session on an exam whose only question was made by
 * createTestQuestion(..., 'single') (correct answer = the option labelled "Option A"). Returns the
 * session id.
 */
export async function createPassedEmployeeSession(examId: string): Promise<string> {
  const token = await getEmployeeApiToken()
  const started = await apiPost<SessionQuestions>(`${BASE}/api/v1/portal/exams/${examId}/sessions`, {}, token)
  if (!started.ok || !started.data?.session_id) {
    throw new Error(`createPassedEmployeeSession: cannot start session: ${started.error ?? started.status}`)
  }
  const { session_id: sessionId, questions } = started.data
  for (const q of questions) {
    const correct = q.options.find((o) => o.text === 'Option A')
    if (!correct) throw new Error(`createPassedEmployeeSession: question ${q.id} has no "Option A" option`)
    const res = await fetch(`${BASE}/api/v1/portal/sessions/${sessionId}/answers/${q.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify({ selected_option_ids: [correct.id], text_answer: null, time_spent_seconds: 1 }),
    })
    if (!res.ok) throw new Error(`createPassedEmployeeSession: saving answer failed (HTTP ${res.status})`)
  }
  await submitSession(token, sessionId)
  return sessionId
}

export async function deleteTestExam(adminToken: string, examId: string): Promise<void> {
  await fetch(`${BASE}/api/v1/exams/${examId}`, {
    method: 'DELETE',
    headers: { Authorization: `Bearer ${adminToken}` },
  })
}

export interface SeedData {
  adminToken: string
  employeeId: string
  mixedExamId: string
  shortTextExamId: string
}

let cachedSeedData: SeedData | null = null

export async function getSeedData(): Promise<SeedData> {
  if (cachedSeedData) return cachedSeedData
  const AUTH_DIR_PATH = path.join(path.dirname(fileURLToPath(import.meta.url)), '..', '..', '.auth')
  const tokenPath = path.join(AUTH_DIR_PATH, 'token.txt')
  const employeeIdPath = path.join(AUTH_DIR_PATH, 'employee-id.txt')
  if (!fs.existsSync(tokenPath)) throw new Error('getSeedData: .auth/token.txt not found — run global setup first')
  if (!fs.existsSync(employeeIdPath)) throw new Error('getSeedData: .auth/employee-id.txt not found — run global setup first')
  let adminToken = fs.readFileSync(tokenPath, 'utf8').trim()
  const employeeId = fs.readFileSync(employeeIdPath, 'utf8').trim()
  if (!employeeId) throw new Error('getSeedData: employee-id.txt is empty')

  // Token from global-setup may have expired (15-min JWT). Re-login if token check fails.
  const tokenCheckOk = await findUserByEmail(adminToken, 'employee@bilimbaga.local').then(id => id !== null).catch(() => false)
  if (!tokenCheckOk) {
    const refreshed = await login(
      process.env.E2E_ADMIN_EMAIL ?? 'admin@bilimbaga.local',
      process.env.E2E_ADMIN_PASS ?? 'Admin1234!',
    )
    if (!refreshed) throw new Error('getSeedData: admin re-login failed')
    adminToken = refreshed
    fs.writeFileSync(tokenPath, adminToken, 'utf8')
  }

  const mixedExamId = await findExam(adminToken, 'E2E Mixed Exam')
  if (!mixedExamId) throw new Error('getSeedData: E2E Mixed Exam not found')
  const shortTextExamId = await findExam(adminToken, 'E2E ShortText Exam')
  if (!shortTextExamId) throw new Error('getSeedData: E2E ShortText Exam not found')
  cachedSeedData = { adminToken, employeeId, mixedExamId, shortTextExamId }
  return cachedSeedData
}

// ---------------------------------------------------------------------------
// Main export
// ---------------------------------------------------------------------------

export async function seedEmployeeFixtures(adminToken: string): Promise<void> {
  console.log('[seed] Starting employee fixture seeding…')

  if (!fs.existsSync(AUTH_DIR)) fs.mkdirSync(AUTH_DIR, { recursive: true })

  // 1. Ensure employee user exists; returns the password to use for login
  const employeePassword = await createEmployee(adminToken)

  // 2. Login as employee to get token
  const employeeToken = await login('employee@bilimbaga.local', employeePassword)
  if (!employeeToken) {
    throw new Error('[seed] Failed to login as employee after creation')
  }
  console.log('[seed] Employee login successful')

  // Save raw token
  fs.writeFileSync(EMPLOYEE_TOKEN_PATH, employeeToken, 'utf8')

  // 3. Get employee user ID
  const employeeId = await findUserByEmail(adminToken, 'employee@bilimbaga.local')
  if (!employeeId) {
    throw new Error('[seed] Could not find employee user ID')
  }
  console.log('[seed] Employee user ID:', employeeId)
  fs.writeFileSync(EMPLOYEE_ID_PATH, employeeId, 'utf8')

  // 4. Create or find "E2E Mixed Exam"
  let mixedExamId = await findExam(adminToken, 'E2E Mixed Exam')
  if (!mixedExamId) {
    console.log('[seed] Creating E2E Mixed Exam…')
    mixedExamId = await createExam(adminToken, 'E2E Mixed Exam', 'warn')
  } else {
    console.log('[seed] E2E Mixed Exam already exists')
  }

  // Add question rules if missing (idempotent: only adds when exam has 0 rules)
  const mixedRuleCount = await getExamRuleCount(adminToken, mixedExamId)
  if (mixedRuleCount === 0) {
    console.log('[seed] E2E Mixed Exam has no rules — adding questions…')
    const singleId = await createQuestion(adminToken, 'single', 'E2E Single Choice Question?', [
      { text: 'Correct Answer', is_correct: true },
      { text: 'Wrong Answer', is_correct: false },
      { text: 'Also Wrong', is_correct: false },
    ])
    const multipleId = await createQuestion(adminToken, 'multiple', 'E2E Multiple Choice Question?', [
      { text: 'Correct A', is_correct: true },
      { text: 'Correct B', is_correct: true },
      { text: 'Wrong C', is_correct: false },
    ])
    const trueFalseId = await createQuestion(adminToken, 'truefalse', 'E2E True/False: Is sky blue?', [
      { text: 'True', is_correct: true },
      { text: 'False', is_correct: false },
    ])
    const likertId = await createQuestion(adminToken, 'likert', 'E2E Likert: Rate your agreement.')
    const shortTextId = await createQuestion(adminToken, 'shorttext', 'E2E Short Text: Describe water in one word.')

    await addRule(adminToken, mixedExamId, singleId, 1)
    await addRule(adminToken, mixedExamId, multipleId, 2)
    await addRule(adminToken, mixedExamId, trueFalseId, 3)
    await addRule(adminToken, mixedExamId, likertId, 4)
    await addRule(adminToken, mixedExamId, shortTextId, 5)

    console.log('[seed] E2E Mixed Exam questions added')
  } else {
    console.log(`[seed] E2E Mixed Exam already has ${mixedRuleCount} rules — skipping question creation`)
  }

  await publishExam(adminToken, mixedExamId)
  await assignExam(adminToken, mixedExamId, employeeId)
  console.log('[seed] E2E Mixed Exam published and assigned')

  // 4b. Repair stale Mixed Exam session: if an in_progress session with 0 questions
  // exists (e.g., from a previous seed run before rules were added), submit it so
  // exam-taking tests can create a fresh session with proper questions.
  const mixedOpenId = await getOpenSessionId(employeeToken, mixedExamId)
  if (mixedOpenId) {
    const mixedState = await getSession(employeeToken, mixedOpenId)
    if (mixedState && mixedState.questions.length === 0) {
      console.log('[seed] Mixed Exam has stale session with 0 questions — force-submitting it')
      await submitSession(employeeToken, mixedOpenId).catch((e) => {
        console.log('[seed] Could not submit stale Mixed Exam session:', e)
      })
    }
  }

  // 5. Create or find "E2E ShortText Exam" (for grading queue tests)
  let shortTextExamId = await findExam(adminToken, 'E2E ShortText Exam')
  if (!shortTextExamId) {
    console.log('[seed] Creating E2E ShortText Exam…')
    shortTextExamId = await createExam(adminToken, 'E2E ShortText Exam', 'log')
  } else {
    console.log('[seed] E2E ShortText Exam already exists')
  }

  const shortTextRuleCount = await getExamRuleCount(adminToken, shortTextExamId)
  if (shortTextRuleCount === 0) {
    console.log('[seed] E2E ShortText Exam has no rules — adding question…')
    const shortTextId2 = await createQuestion(
      adminToken,
      'shorttext',
      'E2E ShortText: What is the capital of Kazakhstan?',
    )
    await addRule(adminToken, shortTextExamId, shortTextId2, 1)
    console.log('[seed] E2E ShortText Exam question added')
  } else {
    console.log(`[seed] E2E ShortText Exam already has ${shortTextRuleCount} rules — skipping`)
  }

  await publishExam(adminToken, shortTextExamId)
  await assignExam(adminToken, shortTextExamId, employeeId)
  console.log('[seed] E2E ShortText Exam published and assigned')

  // 6. Start and submit a session for the ShortText Exam (so it appears in grading queue)
  const sessionId = await startSession(employeeToken, shortTextExamId)
  if (sessionId) {
    await submitSession(employeeToken, sessionId)
    console.log('[seed] ShortText session submitted — will appear in grading queue')
  }

  // 7. Save employee storage state — do a real browser-based login so the
  //    HTTP-only refresh cookie is captured in the storageState. Without the
  //    cookie the token cannot be refreshed when it expires during the test run.
  const browser = await chromium.launch()
  const context = await browser.newContext({ baseURL: APP_URL })
  const page = await context.newPage()
  await page.goto(`${APP_URL}/login`)

  // Browser-fetch login with credentials:include to set the refresh cookie.
  const browserLoginResult = await page.evaluate(
    async ({ email, pass }: { email: string; pass: string }) => {
      const r = await fetch('/api/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password: pass }),
        credentials: 'include',
      })
      const json = await r.json()
      return { ok: r.ok, token: (json.data as { access_token?: string })?.access_token ?? null }
    },
    { email: 'employee@bilimbaga.local', pass: employeePassword },
  )

  // Use the freshly-issued token if available; fall back to node-fetched one.
  const finalToken = browserLoginResult.ok && browserLoginResult.token
    ? browserLoginResult.token
    : employeeToken
  if (!browserLoginResult.ok) {
    console.warn('[seed] Browser-based employee login failed — falling back to node token')
  }

  await page.evaluate((token: string) => {
    localStorage.setItem('__e2e_access_token__', token)
  }, finalToken)
  // Set RU locale for the E2E run.
  await page.evaluate(() => {
    localStorage.setItem('i18n-lang', 'ru')
  })
  await context.storageState({ path: EMPLOYEE_STORAGE_STATE })
  await browser.close()

  console.log('[seed] Employee storage state saved to', EMPLOYEE_STORAGE_STATE)
  console.log('[seed] Employee fixture seeding complete ✓')
}
