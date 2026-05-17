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
import path from 'path'
import { fileURLToPath } from 'url'
import fs from 'fs'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const AUTH_DIR = path.join(__dirname, '..', '..', '.auth')
export const EMPLOYEE_STORAGE_STATE = path.join(AUTH_DIR, 'employee.json')
const EMPLOYEE_TOKEN_PATH = path.join(AUTH_DIR, 'employee-token.txt')

const BASE = 'http://localhost:8080'

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
}

async function createEmployee(adminToken: string): Promise<void> {
  // Try login first (idempotency)
  const token = await login('employee@bilimbaga.local', 'Employee1234!')
  if (token) {
    console.log('[seed] Employee already exists — skipping creation')
    return
  }
  console.log('[seed] Creating employee user…')
  const result = await apiPost<UserData>(
    `${BASE}/api/v1/users`,
    {
      email: 'employee@bilimbaga.local',
      password: 'Employee1234!',
      full_name: 'E2E Employee',
      role: 'employee',
    },
    adminToken,
  )
  if (!result.ok && result.error !== 'DUPLICATE_EMAIL') {
    throw new Error(`Failed to create employee user: ${result.error}`)
  }
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
  const res = await apiGet<ExamListResponse>(`${BASE}/api/v1/exams?per_page=100`, adminToken)
  if (!res.ok || !res.data) return null
  const found = res.data.items.find((e) => e.title === title)
  return found?.id ?? null
}

interface QuestionData {
  id: string
}

async function createQuestion(
  adminToken: string,
  type: string,
  stem: string,
  options?: Array<{ text: string; is_correct: boolean }>,
): Promise<string> {
  const translations: Record<string, { stem: string; explanation: string }> = {
    en: { stem, explanation: '' },
  }

  let answer_options: Array<{
    sort_order: number
    is_correct: boolean
    likert_weight: number | null
    likert_polarity: string | null
    translations: Record<string, { body: string }>
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
      translations: { en: { body: o.text } },
    }))
  } else if (type === 'likert') {
    answer_options = [
      { sort_order: 1, is_correct: false, likert_weight: 1, likert_polarity: 'positive', translations: { en: { body: 'Strongly Agree' } } },
      { sort_order: 2, is_correct: false, likert_weight: 2, likert_polarity: 'positive', translations: { en: { body: 'Agree' } } },
      { sort_order: 3, is_correct: false, likert_weight: 3, likert_polarity: 'positive', translations: { en: { body: 'Neutral' } } },
      { sort_order: 4, is_correct: false, likert_weight: 4, likert_polarity: 'negative', translations: { en: { body: 'Disagree' } } },
      { sort_order: 5, is_correct: false, likert_weight: 5, likert_polarity: 'negative', translations: { en: { body: 'Strongly Disagree' } } },
    ]
  }

  const result = await apiPost<QuestionData>(
    `${BASE}/api/v1/questions`,
    {
      type,
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
  return result.data.id
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

async function assignExam(adminToken: string, examId: string, userId: string): Promise<void> {
  const result = await apiPost<AssignmentData>(
    `${BASE}/api/v1/exams/${examId}/assignments`,
    { assignee_type: 'user', assignee_id: userId, deadline: null },
    adminToken,
  )
  if (!result.ok && result.error !== 'ASSIGNMENT_ALREADY_EXISTS' && result.error !== 'assignmentAlreadyExists') {
    throw new Error(`Failed to assign exam: ${result.error}`)
  }
}

async function publishExam(adminToken: string, examId: string): Promise<void> {
  const result = await apiPost<ExamData>(
    `${BASE}/api/v1/exams/${examId}/publish`,
    {},
    adminToken,
  )
  if (!result.ok && result.error !== 'examNotDraft') {
    throw new Error(`Failed to publish exam: ${result.error}`)
  }
}

interface ExamCreated {
  id: string
}

async function createExam(adminToken: string, title: string, onTabSwitch: string): Promise<string> {
  const result = await apiPost<ExamCreated>(
    `${BASE}/api/v1/exams`,
    {
      title,
      description: 'E2E test exam — created by seed fixture',
      time_limit_minutes: 60,
      passing_score_pct: 70,
      max_attempts: 5,
      available_from: null,
      available_until: null,
      shuffle_questions: false,
      shuffle_options: false,
      show_answers: 'after_completion',
      on_tab_switch: onTabSwitch,
      certificate_enabled: false,
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
  const res = await apiGet<UserListResponse>(
    `${BASE}/api/v1/users?per_page=200`,
    adminToken,
  )
  if (!res.ok || !res.data) return null
  const found = res.data.items.find((u) => u.email === email)
  return found?.id ?? null
}

interface SessionData {
  session_id: string
}

async function startSession(employeeToken: string, examId: string): Promise<string | null> {
  const result = await apiPost<SessionData>(`${BASE}/api/v1/sessions`, { exam_id: examId }, employeeToken)
  if (!result.ok) {
    if (result.error === 'sessionAlreadyOpen' || result.error === 'SESSION_ALREADY_OPEN') {
      console.log('[seed] Session already open for this exam — skipping session creation')
      return null
    }
    throw new Error(`Failed to start session: ${result.error}`)
  }
  return result.data?.session_id ?? null
}

async function submitSession(employeeToken: string, sessionId: string): Promise<void> {
  const result = await apiPost<unknown>(`${BASE}/api/v1/sessions/${sessionId}/submit`, {}, employeeToken)
  if (!result.ok) {
    throw new Error(`Failed to submit session: ${result.error}`)
  }
}

// ---------------------------------------------------------------------------
// Main export
// ---------------------------------------------------------------------------

export async function seedEmployeeFixtures(adminToken: string): Promise<void> {
  console.log('[seed] Starting employee fixture seeding…')

  if (!fs.existsSync(AUTH_DIR)) fs.mkdirSync(AUTH_DIR, { recursive: true })

  // 1. Ensure employee user exists
  await createEmployee(adminToken)

  // 2. Login as employee to get token
  const employeeToken = await login('employee@bilimbaga.local', 'Employee1234!')
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

  // 4. Create or find "E2E Mixed Exam"
  let mixedExamId = await findExam(adminToken, 'E2E Mixed Exam')
  if (!mixedExamId) {
    console.log('[seed] Creating E2E Mixed Exam…')
    mixedExamId = await createExam(adminToken, 'E2E Mixed Exam', 'warn')

    // Create one question per type
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
    console.log('[seed] E2E Mixed Exam already exists, skipping question creation')
  }

  await assignExam(adminToken, mixedExamId, employeeId)
  await publishExam(adminToken, mixedExamId)
  console.log('[seed] E2E Mixed Exam assigned and published')

  // 5. Create or find "E2E ShortText Exam" (for grading queue tests)
  let shortTextExamId = await findExam(adminToken, 'E2E ShortText Exam')
  if (!shortTextExamId) {
    console.log('[seed] Creating E2E ShortText Exam…')
    shortTextExamId = await createExam(adminToken, 'E2E ShortText Exam', 'log')

    const shortTextId2 = await createQuestion(
      adminToken,
      'shorttext',
      'E2E ShortText: What is the capital of Kazakhstan?',
    )
    await addRule(adminToken, shortTextExamId, shortTextId2, 1)
    console.log('[seed] E2E ShortText Exam question added')
  } else {
    console.log('[seed] E2E ShortText Exam already exists, skipping question creation')
  }

  await assignExam(adminToken, shortTextExamId, employeeId)
  await publishExam(adminToken, shortTextExamId)
  console.log('[seed] E2E ShortText Exam assigned and published')

  // 6. Start and submit a session for the ShortText Exam (so it appears in grading queue)
  const sessionId = await startSession(employeeToken, shortTextExamId)
  if (sessionId) {
    await submitSession(employeeToken, sessionId)
    console.log('[seed] ShortText session submitted — will appear in grading queue')
  }

  // 7. Save employee storage state (mirrors global-setup.ts pattern)
  const browser = await chromium.launch()
  const context = await browser.newContext({ baseURL: 'http://localhost:5173' })
  const page = await context.newPage()
  await page.goto('http://localhost:5173/login')
  await page.evaluate((token: string) => {
    localStorage.setItem('__e2e_access_token__', token)
  }, employeeToken)
  await context.storageState({ path: EMPLOYEE_STORAGE_STATE })
  await browser.close()

  console.log('[seed] Employee storage state saved to', EMPLOYEE_STORAGE_STATE)
  console.log('[seed] Employee fixture seeding complete ✓')
}
