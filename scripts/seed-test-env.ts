/**
 * seed-test-env.ts
 *
 * Populates a test environment (E2E_API_URL, required) with realistic test data covering
 * all business processes in the BilimBaga platform.
 *
 * Run: E2E_API_URL=http://localhost:8080 npx tsx scripts/seed-test-env.ts
 *   from the repo root (requires Node 18+, tsx installed globally or via npx)
 *
 * Idempotent: safe to run multiple times; skips already-existing data.
 */

import { checkTarget } from './lib/target-guard'

// E2E_API_URL is REQUIRED (no default): this script WRITES seed data. The URL is parsed and checked against
// an allowlist (localhost, 127.0.0.1, ::1, *.localhost, bilimbaga-qa.ai-dala.com) on the normalised hostname;
// anything else, including the customer demo bilimbaga-test.ai-dala.com, is refused unless
// ALLOW_PROTECTED_HOST=1 (user approval only; swarm roles must never set it).
if (!process.env.E2E_API_URL) {
  console.error('E2E_API_URL is required (e.g. E2E_API_URL=http://localhost:8080). Refusing to run without an explicit target.')
  process.exit(1)
}
const targetDecision = checkTarget('E2E_API_URL', process.env.E2E_API_URL, process.env)
if (!targetDecision.ok) {
  console.error(`Refusing to seed: ${targetDecision.reason}`)
  process.exit(1)
}
if (targetDecision.warning) console.error(targetDecision.warning)
const BASE = targetDecision.url as string
const ADMIN_EMAIL = 'admin@bilimbaga.local'
const ADMIN_INITIAL_PASS = 'Admin1234!'
const ADMIN_KNOWN_PASS = 'Admin2024!'
// Password set by E2E full-walkthrough tests (force_password_change flow)
const ADMIN_E2E_PASS = 'E2eAdmin2024!'
const EMPLOYEE_PASS = 'TestPass2024!'

// ─── Low-level helpers ───────────────────────────────────────────────────────

const RATE_LIMIT_MAX_RETRIES = 5
const RATE_LIMIT_BASE_DELAY_MS = 500

function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms))
}

/** Retries the given fetch on HTTP 429 with exponential backoff before giving up. */
async function fetchWithRateLimitRetry(url: string, init: RequestInit): Promise<Response> {
  for (let attempt = 0; ; attempt++) {
    const res = await fetch(url, init)
    if (res.status !== 429 || attempt >= RATE_LIMIT_MAX_RETRIES) return res
    const retryAfterHeader = res.headers.get('Retry-After')
    const delayMs = retryAfterHeader
      ? Number(retryAfterHeader) * 1000
      : RATE_LIMIT_BASE_DELAY_MS * 2 ** attempt
    warn(`Rate limited (429) on ${url} — retrying in ${delayMs}ms (attempt ${attempt + 1}/${RATE_LIMIT_MAX_RETRIES})`)
    await sleep(delayMs)
  }
}

async function apiPost<T>(
  url: string,
  body: unknown,
  token?: string,
): Promise<{ ok: boolean; status: number; data: T | null; error: string | null }> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  if (token) headers['Authorization'] = `Bearer ${token}`
  const res = await fetchWithRateLimitRetry(url, { method: 'POST', headers, body: JSON.stringify(body) })
  const json = (await res.json()) as { data: T; error: { code: string; message: string } | null }
  return { ok: res.ok, status: res.status, data: json.data ?? null, error: json.error?.code ?? null }
}

async function apiPut<T>(
  url: string,
  body: unknown,
  token: string,
): Promise<{ ok: boolean; status: number; data: T | null; error: string | null }> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` }
  const res = await fetchWithRateLimitRetry(url, { method: 'PUT', headers, body: JSON.stringify(body) })
  const json = (await res.json()) as { data: T; error: { code: string; message: string } | null }
  return { ok: res.ok, status: res.status, data: json.data ?? null, error: json.error?.code ?? null }
}

async function apiGet<T>(
  url: string,
  token: string,
): Promise<{ ok: boolean; status: number; data: T | null; error: string | null }> {
  const res = await fetchWithRateLimitRetry(url, { headers: { Authorization: `Bearer ${token}` } })
  const json = (await res.json()) as { data: T; error: { code: string; message: string } | null }
  return { ok: res.ok, status: res.status, data: json.data ?? null, error: json.error?.code ?? null }
}

function log(msg: string) { console.log(`[seed] ${msg}`) }
function warn(msg: string) { console.warn(`[seed] ⚠ ${msg}`) }

// ─── Auth ────────────────────────────────────────────────────────────────────

interface LoginData { access_token: string; user: { id: string; force_password_change: boolean } }

interface LoginResult { token: string; forceChange: boolean }
type LoginOutcome = LoginResult | { locked: true; lockedUntil: string } | null

async function login(email: string, pass: string): Promise<LoginOutcome> {
  const r = await apiPost<LoginData>(`${BASE}/api/v1/auth/login`, { email, password: pass })
  if (r.ok && r.data) return { token: r.data.access_token, forceChange: r.data.user.force_password_change }
  if (r.error === 'ACCOUNT_LOCKED') {
    // Parse lock expiry from the raw response message if available
    const res = await fetch(`${BASE}/api/v1/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password: pass }),
    }).catch(() => null)
    const msg = res ? ((await res.json().catch(() => ({}))) as { error?: { message?: string } })?.error?.message ?? '' : ''
    const match = msg.match(/after (.+)$/)
    return { locked: true, lockedUntil: match ? match[1] : 'unknown time' }
  }
  return null
}

function isLoginResult(o: LoginOutcome): o is LoginResult {
  return o !== null && !('locked' in o)
}

/** Login for non-admin accounts — returns token or null (locked treated as failure). */
async function loginUser(email: string, pass: string): Promise<LoginResult | null> {
  const outcome = await login(email, pass)
  return isLoginResult(outcome) ? outcome : null
}

async function tryLoginWithPassword(email: string, pass: string, label: string): Promise<LoginResult | null> {
  const outcome = await login(email, pass)
  if (isLoginResult(outcome)) return outcome
  if (outcome && 'locked' in outcome) {
    throw new Error(
      `Admin account is locked until ${outcome.lockedUntil}. ` +
      `Deploy migration 030_reset_admin_password to unlock it, then re-run this script.`,
    )
  }
  warn(`Password "${label}" did not work`)
  return null
}

/**
 * ISS-160: a flagged admin (force_password_change) is rejected with 403 PASSWORD_CHANGE_REQUIRED on
 * every route but change-password, so change it with the login token before any other call. The
 * password stays unchanged when it already is the known one (the backend does not reject reuse).
 */
async function changeForcedAdminPassword(result: LoginResult, currentPass: string): Promise<string> {
  const changed = await apiPost<unknown>(
    `${BASE}/api/v1/auth/change-password`,
    { current_password: currentPass, new_password: currentPass },
    result.token,
  )
  if (!changed.ok) throw new Error(`Failed to clear forced admin password change: ${changed.error}`)
  log('Admin forced password change completed')
  return result.token
}

async function ensureAdminToken(): Promise<string> {
  // Probe for lock before burning attempts
  const lockCheck = await login(ADMIN_EMAIL, '___probe___')
  if (lockCheck && 'locked' in lockCheck) {
    throw new Error(
      `Admin account is locked until ${lockCheck.lockedUntil}. ` +
      `Deploy migration 030_reset_admin_password to unlock it, then re-run this script.`,
    )
  }

  // Operator-provided admin password (BOOTSTRAP_ADMIN_PASSWORD on the API / E2E_ADMIN_PASS here):
  // the deployment no longer relies on the migration-shipped default (ISS-150/ISS-152).
  const envPass = process.env.E2E_ADMIN_PASS
  if (envPass && envPass !== ADMIN_INITIAL_PASS) {
    const envResult = await tryLoginWithPassword(ADMIN_EMAIL, envPass, 'E2E_ADMIN_PASS')
    if (envResult) {
      log('Admin login OK (E2E_ADMIN_PASS)')
      return envResult.forceChange ? changeForcedAdminPassword(envResult, envPass) : envResult.token
    }
  }

  // Try known post-seed password first (idempotent re-run)
  let result = await tryLoginWithPassword(ADMIN_EMAIL, ADMIN_KNOWN_PASS, 'Admin2024!')
  if (result) {
    log('Admin login OK (known password)')
    return result.forceChange ? changeForcedAdminPassword(result, ADMIN_KNOWN_PASS) : result.token
  }

  // Try E2E walkthrough password (set when force_password_change flow runs)
  result = await tryLoginWithPassword(ADMIN_EMAIL, ADMIN_E2E_PASS, 'E2eAdmin2024!')
  if (result) {
    log('Admin login OK (E2E password) — normalising to known password')
    const changed = await apiPost<unknown>(
      `${BASE}/api/v1/auth/change-password`,
      { current_password: ADMIN_E2E_PASS, new_password: ADMIN_KNOWN_PASS },
      result.token,
    )
    if (!changed.ok) throw new Error(`Failed to change admin password: ${changed.error}`)
    const next = await tryLoginWithPassword(ADMIN_EMAIL, ADMIN_KNOWN_PASS, 'Admin2024!')
    if (!next) throw new Error('Admin login failed after password normalisation')
    log('Admin password normalised')
    return next.token
  }

  // Try initial migration password
  result = await tryLoginWithPassword(ADMIN_EMAIL, ADMIN_INITIAL_PASS, 'Admin1234!')
  if (!result) {
    throw new Error(
      `Admin login failed with all known passwords (Admin2024!, E2eAdmin2024!, Admin1234!). ` +
      `The admin password was changed to an unknown value. ` +
      `Deploy migration 030_reset_admin_password to reset it, then re-run this script.`,
    )
  }
  log('Admin login OK (initial password) — changing to known password')

  const changed = await apiPost<unknown>(
    `${BASE}/api/v1/auth/change-password`,
    { current_password: ADMIN_INITIAL_PASS, new_password: ADMIN_KNOWN_PASS },
    result.token,
  )
  if (!changed.ok) throw new Error(`Failed to change admin password: ${changed.error}`)

  result = await tryLoginWithPassword(ADMIN_EMAIL, ADMIN_KNOWN_PASS, 'Admin2024!')
  if (!result) throw new Error('Admin login failed after password change')
  log('Admin password changed successfully')
  return result.token
}

// ─── Roles ───────────────────────────────────────────────────────────────────

interface Role { id: string; name: string }

async function getRoles(token: string): Promise<Record<string, string>> {
  const r = await apiGet<Role[]>(`${BASE}/api/v1/users/roles`, token)
  if (!r.ok || !r.data) throw new Error('Failed to fetch roles')
  const map: Record<string, string> = {}
  for (const role of r.data) map[role.name] = role.id
  log(`Roles: ${Object.keys(map).join(', ')}`)
  return map
}

// ─── Departments ─────────────────────────────────────────────────────────────

interface Department { id: string; name: string; children?: Department[] }

async function listDepartments(token: string): Promise<Department[]> {
  const r = await apiGet<Department[]>(`${BASE}/api/v1/departments`, token)
  return r.data ?? []
}

function flattenDepts(depts: Department[]): Department[] {
  const flat: Department[] = []
  for (const d of depts) {
    flat.push(d)
    if (d.children) flat.push(...flattenDepts(d.children))
  }
  return flat
}

async function ensureDepartment(token: string, name: string, existing: Department[]): Promise<string> {
  const found = existing.find(d => d.name === name)
  if (found) {
    log(`Dept "${name}" already exists (${found.id})`)
    return found.id
  }
  const r = await apiPost<Department>(`${BASE}/api/v1/departments`, { name }, token)
  if (!r.ok || !r.data) throw new Error(`Failed to create dept "${name}": ${r.error}`)
  log(`Created dept "${name}" → ${r.data.id}`)
  return r.data.id
}

// ─── Users ───────────────────────────────────────────────────────────────────

interface UserRecord { id: string; email: string; full_name: string; temporary_password?: string }
interface UserList { items: Array<{ id: string; email: string }> }

async function findUserByEmail(token: string, email: string): Promise<string | null> {
  const r = await apiGet<UserList>(`${BASE}/api/v1/users?per_page=200`, token)
  if (!r.ok || !r.data) return null
  return r.data.items.find(u => u.email === email)?.id ?? null
}

async function ensureUser(
  token: string,
  email: string,
  fullName: string,
  roleId: string,
  departmentId?: string,
): Promise<string> {
  // Check idempotency
  const existing = await findUserByEmail(token, email)
  if (existing) {
    log(`User "${email}" already exists (${existing})`)
    return existing
  }

  const body: Record<string, unknown> = { email, full_name: fullName, role_id: roleId }
  if (departmentId) body.department_id = departmentId

  const r = await apiPost<UserRecord>(`${BASE}/api/v1/users`, body, token)
  if (!r.ok || !r.data) {
    if (r.error === 'DUPLICATE_EMAIL') {
      const id = await findUserByEmail(token, email)
      if (id) return id
    }
    throw new Error(`Failed to create user "${email}": ${r.error}`)
  }

  const userId = r.data.id
  const tempPass = r.data.temporary_password
  if (!tempPass) {
    warn(`No temp password returned for ${email} — skipping password set`)
    return userId
  }

  // Set known password
  const loginResult = await loginUser(email, tempPass)
  if (!loginResult) throw new Error(`Could not login as ${email} with temp pass`)
  const changed = await apiPost<unknown>(
    `${BASE}/api/v1/auth/change-password`,
    { current_password: tempPass, new_password: EMPLOYEE_PASS },
    loginResult.token,
  )
  if (!changed.ok) warn(`Could not change password for ${email}: ${changed.error}`)

  log(`Created user "${email}" → ${userId}`)
  return userId
}

// ─── Categories ──────────────────────────────────────────────────────────────

interface Category { id: string; name: string; children?: Category[] }

async function getCategories(token: string): Promise<Category[]> {
  const r = await apiGet<Category[]>(`${BASE}/api/v1/categories`, token)
  return r.data ?? []
}

function flattenCategories(cats: Category[]): Category[] {
  const flat: Category[] = []
  for (const c of cats) {
    flat.push(c)
    if (c.children) flat.push(...flattenCategories(c.children))
  }
  return flat
}

async function ensureCategory(token: string, name: string, existing: Category[]): Promise<string> {
  const found = existing.find(c => c.name === name)
  if (found) {
    log(`Category "${name}" already exists (${found.id})`)
    return found.id
  }
  const r = await apiPost<Category>(`${BASE}/api/v1/categories`, { name }, token)
  if (!r.ok || !r.data) throw new Error(`Failed to create category "${name}": ${r.error}`)
  log(`Created category "${name}" → ${r.data.id}`)
  return r.data.id
}

// ─── Questions ───────────────────────────────────────────────────────────────

interface QuestionCreated { id: string }

// Maps the script's descriptive question type labels to the API's wire format.
const QUESTION_TYPE_WIRE: Record<string, string> = {
  single_choice: 'single',
  multiple_choice: 'multiple',
  true_false: 'truefalse',
  likert: 'likert',
  short_text: 'shorttext',
}

async function createQuestion(
  token: string,
  type: 'single_choice' | 'multiple_choice' | 'true_false' | 'likert' | 'short_text',
  stem: string,
  categoryId: string,
  difficulty: 'easy' | 'medium' | 'hard',
  options?: Array<{ text: string; is_correct: boolean }>,
): Promise<string> {
  let answer_options: unknown[] = []

  if (type === 'single_choice' || type === 'multiple_choice' || type === 'true_false') {
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
      { sort_order: 3, is_correct: false, likert_weight: 3, likert_polarity: null, translations: { en: { text: 'Neutral' } } },
      { sort_order: 4, is_correct: false, likert_weight: 4, likert_polarity: 'negative', translations: { en: { text: 'Disagree' } } },
      { sort_order: 5, is_correct: false, likert_weight: 5, likert_polarity: 'negative', translations: { en: { text: 'Strongly Disagree' } } },
    ]
  } else {
    // short_text: no options
    answer_options = []
  }

  const r = await apiPost<QuestionCreated>(
    `${BASE}/api/v1/questions`,
    {
      type: QUESTION_TYPE_WIRE[type],
      category_id: categoryId,
      difficulty,
      default_locale: 'en',
      translations: { en: { stem, explanation: '' } },
      answer_options,
    },
    token,
  )
  if (!r.ok || !r.data?.id) throw new Error(`Failed to create question "${stem}": ${r.error}`)
  const questionId = r.data.id

  // draft → review → active
  const toReview = await apiPost<unknown>(`${BASE}/api/v1/questions/${questionId}/status`, { status: 'review' }, token)
  if (!toReview.ok) throw new Error(`Question ${questionId} draft→review failed: ${toReview.error}`)
  const toActive = await apiPost<unknown>(`${BASE}/api/v1/questions/${questionId}/status`, { status: 'active' }, token)
  if (!toActive.ok) throw new Error(`Question ${questionId} review→active failed: ${toActive.error}`)

  return questionId
}

// ─── Exams ───────────────────────────────────────────────────────────────────

interface ExamCreated { id: string; title: string }
interface ExamList { items: Array<{ id: string; title: string }> }

async function findExam(token: string, title: string): Promise<string | null> {
  const r = await apiGet<ExamList>(`${BASE}/api/v1/exams?per_page=100`, token)
  return r.data?.items.find(e => e.title === title)?.id ?? null
}

async function ensureExam(
  token: string,
  config: {
    title: string
    description: string
    timeLimitMinutes: number
    passingScorePct: number
    maxAttempts: number
    shuffleQuestions: boolean
    shuffleOptions: boolean
    showAnswers: 'after_completion' | 'never' | 'immediately'
    onTabSwitch: 'log' | 'submit' | 'warn'
    certificateEnabled: boolean
  },
): Promise<string> {
  const existing = await findExam(token, config.title)
  if (existing) {
    log(`Exam "${config.title}" already exists (${existing})`)
    return existing
  }
  const r = await apiPost<ExamCreated>(`${BASE}/api/v1/exams`, {
    title: config.title,
    description: config.description,
    time_limit_minutes: config.timeLimitMinutes,
    passing_score_pct: config.passingScorePct,
    max_attempts: config.maxAttempts,
    available_from: null,
    available_until: null,
    shuffle_questions: config.shuffleQuestions,
    shuffle_options: config.shuffleOptions,
    show_answers: config.showAnswers,
    on_tab_switch: config.onTabSwitch,
    certificate_enabled: config.certificateEnabled,
  }, token)
  if (!r.ok || !r.data?.id) throw new Error(`Failed to create exam "${config.title}": ${r.error}`)
  log(`Created exam "${config.title}" → ${r.data.id}`)
  return r.data.id
}

async function addQuestionRule(token: string, examId: string, questionId: string, sortOrder: number): Promise<void> {
  const r = await apiPost<unknown>(`${BASE}/api/v1/exams/${examId}/rules`, {
    mode: 'manual',
    count: 1,
    sort_order: sortOrder,
    questions: [{ question_id: questionId, sort_order: 1 }],
  }, token)
  if (!r.ok) throw new Error(`Failed to add rule for question ${questionId}: ${r.error}`)
}

async function publishExam(token: string, examId: string): Promise<void> {
  const r = await apiPost<unknown>(`${BASE}/api/v1/exams/${examId}/publish`, {}, token)
  if (!r.ok && r.error !== 'EXAM_ALREADY_ACTIVE') throw new Error(`Failed to publish exam ${examId}: ${r.error}`)
}

// ─── Assignments ─────────────────────────────────────────────────────────────

interface AssignmentList { items?: Array<{ assignee_id: string | null }> }

async function assignExamToUser(token: string, examId: string, userId: string, deadlineDaysOffset?: number): Promise<void> {
  // Check if already assigned
  const r = await apiGet<AssignmentList>(`${BASE}/api/v1/exams/${examId}/assignments`, token)
  const items = Array.isArray(r.data) ? r.data : (r.data?.items ?? [])
  if (items.some((a: { assignee_id: string | null }) => a.assignee_id === userId)) {
    log(`Exam ${examId} already assigned to ${userId}`)
    return
  }

  let deadline: string | null = null
  if (deadlineDaysOffset !== undefined) {
    const d = new Date()
    d.setDate(d.getDate() + deadlineDaysOffset)
    deadline = d.toISOString()
  }

  const assign = await apiPost<unknown>(`${BASE}/api/v1/exams/${examId}/assign`, {
    assignee_type: 'user',
    assignee_id: userId,
    deadline,
  }, token)
  if (!assign.ok && assign.error !== 'ASSIGNMENT_ALREADY_EXISTS') {
    warn(`Failed to assign exam ${examId} to user ${userId}: ${assign.error}`)
  }
}

// ─── Sessions ────────────────────────────────────────────────────────────────

interface SessionCreated { session_id: string; questions: SessionQuestion[] }
interface SessionQuestion {
  id: string
  type: string
  options: Array<{ id: string; text: string }>
}

interface PortalExam { id: string; open_session_id: string | null }

async function getOpenSession(empToken: string, examId: string): Promise<string | null> {
  const r = await apiGet<PortalExam[]>(`${BASE}/api/v1/portal/exams`, empToken)
  return r.data?.find((e: PortalExam) => e.id === examId)?.open_session_id ?? null
}

async function completeSingleSession(
  empToken: string,
  examId: string,
  answerCorrectly: boolean,
): Promise<string | null> {
  // Start session (or find existing open one)
  let sessionId: string | null = null
  let questions: SessionQuestion[] = []

  const startR = await apiPost<SessionCreated>(`${BASE}/api/v1/portal/exams/${examId}/sessions`, {}, empToken)
  if (startR.ok && startR.data) {
    sessionId = startR.data.session_id
    questions = startR.data.questions
  } else if (startR.error === 'SESSION_ALREADY_OPEN') {
    sessionId = await getOpenSession(empToken, examId)
    if (sessionId) {
      const stateR = await apiGet<{ questions: SessionQuestion[] }>(`${BASE}/api/v1/portal/sessions/${sessionId}`, empToken)
      questions = stateR.data?.questions ?? []
    }
  } else if (startR.error === 'ATTEMPTS_EXHAUSTED') {
    log(`Attempts exhausted for exam ${examId} — skipping`)
    return null
  } else {
    warn(`Could not start session for exam ${examId}: ${startR.error}`)
    return null
  }

  if (!sessionId) return null

  // Answer each question
  for (const q of questions) {
    if (q.type === 'short_text') {
      await apiPut<unknown>(
        `${BASE}/api/v1/portal/sessions/${sessionId}/answers/${q.id}`,
        { selected_option_ids: [], text_answer: answerCorrectly ? 'This is a correct and detailed answer.' : 'idk', time_spent_seconds: 45 },
        empToken,
      )
    } else {
      const correctOption = q.options[0] // first option is always the correct one we defined
      const wrongOption = q.options[q.options.length - 1] // last option is wrong
      const chosenOption = answerCorrectly ? correctOption : wrongOption
      if (chosenOption) {
        await apiPut<unknown>(
          `${BASE}/api/v1/portal/sessions/${sessionId}/answers/${q.id}`,
          { selected_option_ids: [chosenOption.id], text_answer: null, time_spent_seconds: 30 },
          empToken,
        )
      }
    }
  }

  // Submit
  const submitR = await apiPost<unknown>(`${BASE}/api/v1/portal/sessions/${sessionId}/submit`, {}, empToken)
  if (!submitR.ok) warn(`Submit failed for session ${sessionId}: ${submitR.error}`)
  else log(`Session ${sessionId} submitted`)

  return sessionId
}

// ─── Main ────────────────────────────────────────────────────────────────────

async function main(): Promise<void> {
  log('=== BilimBaga Test Environment Seeder ===')
  log(`Target: ${BASE}`)

  // ── 1. Admin auth ──────────────────────────────────────────────────────────
  const adminToken = await ensureAdminToken()

  // ── 2. Departments ─────────────────────────────────────────────────────────
  log('\n── Departments ──')
  const existingDepts = flattenDepts(await listDepartments(adminToken))
  const deptEngineering = await ensureDepartment(adminToken, 'Engineering', existingDepts)
  const deptHR = await ensureDepartment(adminToken, 'Human Resources', existingDepts)
  const deptFinance = await ensureDepartment(adminToken, 'Finance', existingDepts)
  const deptOperations = await ensureDepartment(adminToken, 'Operations', existingDepts)

  // ── 3. Roles ───────────────────────────────────────────────────────────────
  log('\n── Roles ──')
  const roles = await getRoles(adminToken)
  const deptAdminRoleId = roles['department_admin']
  const examinerRoleId = roles['examiner']
  const employeeRoleId = roles['employee']
  if (!deptAdminRoleId || !examinerRoleId || !employeeRoleId) {
    throw new Error(`Missing roles: ${JSON.stringify(roles)}`)
  }

  // ── 4. Users ───────────────────────────────────────────────────────────────
  log('\n── Users ──')

  // Department Admins
  const daEngId = await ensureUser(adminToken, 'da.engineering@bilimbaga-test.local', 'Alex Kim (Eng Admin)', deptAdminRoleId, deptEngineering)
  const daHrId = await ensureUser(adminToken, 'da.hr@bilimbaga-test.local', 'Morgan Lee (HR Admin)', deptAdminRoleId, deptHR)

  // Examiners
  const examiner1Id = await ensureUser(adminToken, 'examiner@bilimbaga-test.local', 'Jamie Chen (Examiner)', examinerRoleId)
  const _examiner2Id = await ensureUser(adminToken, 'examiner2@bilimbaga-test.local', 'Taylor Ramos (Examiner)', examinerRoleId)

  void daEngId; void daHrId; void examiner1Id

  // Employees — Engineering (4)
  const alice = await ensureUser(adminToken, 'alice@bilimbaga-test.local', 'Alice Johnson', employeeRoleId, deptEngineering)
  const bob = await ensureUser(adminToken, 'bob@bilimbaga-test.local', 'Bob Smith', employeeRoleId, deptEngineering)
  const carol = await ensureUser(adminToken, 'carol@bilimbaga-test.local', 'Carol White', employeeRoleId, deptEngineering)
  const dave = await ensureUser(adminToken, 'dave@bilimbaga-test.local', 'Dave Brown', employeeRoleId, deptEngineering)

  // Employees — HR (3)
  const eva = await ensureUser(adminToken, 'eva@bilimbaga-test.local', 'Eva Martinez', employeeRoleId, deptHR)
  const frank = await ensureUser(adminToken, 'frank@bilimbaga-test.local', 'Frank Davis', employeeRoleId, deptHR)
  const grace = await ensureUser(adminToken, 'grace@bilimbaga-test.local', 'Grace Wilson', employeeRoleId, deptHR)

  // Employees — Finance (2)
  const henry = await ensureUser(adminToken, 'henry@bilimbaga-test.local', 'Henry Taylor', employeeRoleId, deptFinance)
  const iris = await ensureUser(adminToken, 'iris@bilimbaga-test.local', 'Iris Anderson', employeeRoleId, deptFinance)

  // Employees — Operations (2)
  const jack = await ensureUser(adminToken, 'jack@bilimbaga-test.local', 'Jack Thompson', employeeRoleId, deptOperations)
  const kate = await ensureUser(adminToken, 'kate@bilimbaga-test.local', 'Kate Harris', employeeRoleId, deptOperations)

  // ── 5. Categories ──────────────────────────────────────────────────────────
  log('\n── Categories ──')
  const existingCats = flattenCategories(await getCategories(adminToken))
  const catSecurity = await ensureCategory(adminToken, 'Security Awareness', existingCats)
  const catSafety = await ensureCategory(adminToken, 'Workplace Safety', existingCats)
  const catLoyalty = await ensureCategory(adminToken, 'Loyalty & Values', existingCats)
  const catHR = await ensureCategory(adminToken, 'HR Compliance', existingCats)
  const catIT = await ensureCategory(adminToken, 'IT Fundamentals', existingCats)

  // ── 6. Questions ───────────────────────────────────────────────────────────
  log('\n── Questions ──')

  // Security Awareness (single_choice, true_false, multiple_choice)
  const qSec1 = await createQuestion(adminToken, 'single_choice',
    'What should you do if you receive a suspicious email asking for your password?',
    catSecurity, 'easy',
    [
      { text: 'Delete it and report to IT security', is_correct: true },
      { text: 'Reply with your password', is_correct: false },
      { text: 'Forward it to colleagues', is_correct: false },
      { text: 'Ignore it and do nothing', is_correct: false },
    ])

  const qSec2 = await createQuestion(adminToken, 'true_false',
    'You should use the same password for all your work accounts to make it easier to remember.',
    catSecurity, 'easy',
    [
      { text: 'True', is_correct: false },
      { text: 'False', is_correct: true },
    ])

  const qSec3 = await createQuestion(adminToken, 'multiple_choice',
    'Which of the following are signs of a phishing email? (Select all that apply)',
    catSecurity, 'medium',
    [
      { text: 'Urgency or threatening language', is_correct: true },
      { text: 'Misspelled sender domain', is_correct: true },
      { text: 'Request for personal credentials', is_correct: true },
      { text: 'Company logo in the header', is_correct: false },
    ])

  const qSec4 = await createQuestion(adminToken, 'single_choice',
    'How often should you update your work password?',
    catSecurity, 'easy',
    [
      { text: 'Every 90 days or when compromised', is_correct: true },
      { text: 'Never — changing passwords is risky', is_correct: false },
      { text: 'Once a year', is_correct: false },
      { text: 'Only when IT forces you to', is_correct: false },
    ])

  const qSec5 = await createQuestion(adminToken, 'short_text',
    'Describe in your own words what "social engineering" means in the context of cybersecurity.',
    catSecurity, 'medium')

  // Workplace Safety
  const qSaf1 = await createQuestion(adminToken, 'single_choice',
    'What is the correct procedure when you discover a fire in the office?',
    catSafety, 'easy',
    [
      { text: 'Activate the nearest fire alarm and evacuate via the emergency exit', is_correct: true },
      { text: 'Try to extinguish it yourself before calling for help', is_correct: false },
      { text: 'Call your manager first and wait for instructions', is_correct: false },
      { text: 'Collect your personal belongings before evacuating', is_correct: false },
    ])

  const qSaf2 = await createQuestion(adminToken, 'true_false',
    'It is acceptable to block emergency exits with boxes for short periods of time.',
    catSafety, 'easy',
    [
      { text: 'True', is_correct: false },
      { text: 'False', is_correct: true },
    ])

  const qSaf3 = await createQuestion(adminToken, 'multiple_choice',
    'Which of the following are required when working with electrical equipment? (Select all that apply)',
    catSafety, 'medium',
    [
      { text: 'Turn off and unplug before servicing', is_correct: true },
      { text: 'Use insulated tools', is_correct: true },
      { text: 'Report damaged cables immediately', is_correct: true },
      { text: 'Work faster to minimize risk exposure time', is_correct: false },
    ])

  const qSaf4 = await createQuestion(adminToken, 'short_text',
    'Explain how you would respond if a colleague was injured in the workplace.',
    catSafety, 'medium')

  // HR Compliance
  const qHR1 = await createQuestion(adminToken, 'single_choice',
    'What is the notice period for resignations as per company policy?',
    catHR, 'easy',
    [
      { text: '30 calendar days', is_correct: true },
      { text: '2 weeks', is_correct: false },
      { text: '60 days', is_correct: false },
      { text: 'No notice required', is_correct: false },
    ])

  const qHR2 = await createQuestion(adminToken, 'true_false',
    'Discussing your salary with colleagues is grounds for immediate dismissal.',
    catHR, 'medium',
    [
      { text: 'True', is_correct: false },
      { text: 'False', is_correct: true },
    ])

  const qHR3 = await createQuestion(adminToken, 'likert',
    'I understand the company\'s code of conduct and how it applies to my daily work.',
    catHR, 'easy')

  const qHR4 = await createQuestion(adminToken, 'multiple_choice',
    'Which of the following constitute workplace harassment? (Select all that apply)',
    catHR, 'medium',
    [
      { text: 'Unwanted physical contact', is_correct: true },
      { text: 'Repeated offensive comments about appearance', is_correct: true },
      { text: 'Excluding someone from team activities based on protected characteristics', is_correct: true },
      { text: 'Providing constructive performance feedback', is_correct: false },
    ])

  const qHR5 = await createQuestion(adminToken, 'short_text',
    'Describe a workplace situation that could be considered a conflict of interest.',
    catHR, 'hard')

  // Loyalty & Values
  const qLoy1 = await createQuestion(adminToken, 'likert',
    'I feel proud to work for this organisation.',
    catLoyalty, 'easy')

  const qLoy2 = await createQuestion(adminToken, 'likert',
    'I would recommend this company as a great place to work.',
    catLoyalty, 'easy')

  const qLoy3 = await createQuestion(adminToken, 'likert',
    'I understand and believe in the company\'s mission and values.',
    catLoyalty, 'easy')

  // IT Fundamentals
  const qIT1 = await createQuestion(adminToken, 'single_choice',
    'What does "VPN" stand for?',
    catIT, 'easy',
    [
      { text: 'Virtual Private Network', is_correct: true },
      { text: 'Very Protected Node', is_correct: false },
      { text: 'Virtual Processing Network', is_correct: false },
      { text: 'Verified Public Network', is_correct: false },
    ])

  const qIT2 = await createQuestion(adminToken, 'multiple_choice',
    'Which of the following are best practices for data backup? (Select all that apply)',
    catIT, 'medium',
    [
      { text: 'Follow the 3-2-1 rule (3 copies, 2 media, 1 offsite)', is_correct: true },
      { text: 'Test restores regularly', is_correct: true },
      { text: 'Encrypt backup files', is_correct: true },
      { text: 'Store all backups on the same server as production', is_correct: false },
    ])

  const qIT3 = await createQuestion(adminToken, 'single_choice',
    'Which port does HTTPS use by default?',
    catIT, 'medium',
    [
      { text: '443', is_correct: true },
      { text: '80', is_correct: false },
      { text: '8080', is_correct: false },
      { text: '22', is_correct: false },
    ])

  const qIT4 = await createQuestion(adminToken, 'true_false',
    'Multi-factor authentication (MFA) significantly reduces the risk of unauthorized access.',
    catIT, 'easy',
    [
      { text: 'True', is_correct: true },
      { text: 'False', is_correct: false },
    ])

  log(`Created ${[qSec1, qSec2, qSec3, qSec4, qSec5, qSaf1, qSaf2, qSaf3, qSaf4, qHR1, qHR2, qHR3, qHR4, qHR5, qLoy1, qLoy2, qLoy3, qIT1, qIT2, qIT3, qIT4].length} questions`)

  // ── 7. Exams ───────────────────────────────────────────────────────────────
  log('\n── Exams ──')

  // Exam A: IT Security Fundamentals (published, certificate, 60 min, 70% pass)
  const examAId = await ensureExam(adminToken, {
    title: 'IT Security Fundamentals',
    description: 'Covers essential cybersecurity concepts every employee must know, including password management, phishing awareness, and safe internet usage.',
    timeLimitMinutes: 60,
    passingScorePct: 70,
    maxAttempts: 3,
    shuffleQuestions: true,
    shuffleOptions: true,
    showAnswers: 'after_completion',
    onTabSwitch: 'warn',
    certificateEnabled: true,
  })
  // Add questions only if exam was just created (check rule count)
  const examAInfo = await apiGet<{ id: string; rules: unknown[] }>(`${BASE}/api/v1/exams/${examAId}`, adminToken)
  if (!examAInfo.data?.rules?.length) {
    let i = 1
    for (const qId of [qSec1, qSec2, qSec3, qSec4, qIT1, qIT4]) {
      await addQuestionRule(adminToken, examAId, qId, i++)
    }
    await publishExam(adminToken, examAId)
    log(`IT Security Fundamentals: 6 questions added & published`)
  } else {
    log(`IT Security Fundamentals: already has rules, skipping`)
  }

  // Exam B: Workplace Safety Certification (published, short text, cert enabled, 45 min, 80%)
  const examBId = await ensureExam(adminToken, {
    title: 'Workplace Safety Certification',
    description: 'Mandatory safety certification covering fire procedures, emergency protocols, and hazard reporting.',
    timeLimitMinutes: 45,
    passingScorePct: 80,
    maxAttempts: 2,
    shuffleQuestions: false,
    shuffleOptions: false,
    showAnswers: 'after_completion',
    onTabSwitch: 'log',
    certificateEnabled: true,
  })
  const examBInfo = await apiGet<{ id: string; rules: unknown[] }>(`${BASE}/api/v1/exams/${examBId}`, adminToken)
  if (!examBInfo.data?.rules?.length) {
    let i = 1
    for (const qId of [qSaf1, qSaf2, qSaf3, qSaf4]) {
      await addQuestionRule(adminToken, examBId, qId, i++)
    }
    await publishExam(adminToken, examBId)
    log(`Workplace Safety Certification: 4 questions added & published (includes short text)`)
  }

  // Exam C: HR Compliance Quiz (published, 30 min, 70%)
  const examCId = await ensureExam(adminToken, {
    title: 'HR Compliance Quiz',
    description: 'Tests understanding of company HR policies, employment law, and conduct standards.',
    timeLimitMinutes: 30,
    passingScorePct: 70,
    maxAttempts: 5,
    shuffleQuestions: true,
    shuffleOptions: false,
    showAnswers: 'after_completion',
    onTabSwitch: 'log',
    certificateEnabled: false,
  })
  const examCInfo = await apiGet<{ id: string; rules: unknown[] }>(`${BASE}/api/v1/exams/${examCId}`, adminToken)
  if (!examCInfo.data?.rules?.length) {
    let i = 1
    for (const qId of [qHR1, qHR2, qHR3, qHR4, qHR5, qSec5]) {
      await addQuestionRule(adminToken, examCId, qId, i++)
    }
    await publishExam(adminToken, examCId)
    log(`HR Compliance Quiz: 6 questions added & published`)
  }

  // Exam D: Employee Loyalty Survey (published, no time limit, Likert only)
  const examDId = await ensureExam(adminToken, {
    title: 'Employee Loyalty & Values Survey',
    description: 'An anonymous survey to measure employee engagement and alignment with company values. Results are used to improve workplace culture.',
    timeLimitMinutes: 999,
    passingScorePct: 0,
    maxAttempts: 1,
    shuffleQuestions: false,
    shuffleOptions: false,
    showAnswers: 'never',
    onTabSwitch: 'log',
    certificateEnabled: false,
  })
  const examDInfo = await apiGet<{ id: string; rules: unknown[] }>(`${BASE}/api/v1/exams/${examDId}`, adminToken)
  if (!examDInfo.data?.rules?.length) {
    let i = 1
    for (const qId of [qLoy1, qLoy2, qLoy3, qHR3]) {
      await addQuestionRule(adminToken, examDId, qId, i++)
    }
    await publishExam(adminToken, examDId)
    log(`Employee Loyalty & Values Survey: 4 questions added & published`)
  }

  // Exam E: IT Advanced Security (DRAFT — left for authoring demo)
  const examEId = await ensureExam(adminToken, {
    title: 'IT Advanced Security Assessment (DRAFT)',
    description: 'Advanced assessment covering incident response, network security, and data governance. Work in progress.',
    timeLimitMinutes: 90,
    passingScorePct: 75,
    maxAttempts: 2,
    shuffleQuestions: true,
    shuffleOptions: true,
    showAnswers: 'never',
    onTabSwitch: 'submit',
    certificateEnabled: true,
  })
  const examEInfo = await apiGet<{ id: string; rules: unknown[]; status: string }>(`${BASE}/api/v1/exams/${examEId}`, adminToken)
  if (!examEInfo.data?.rules?.length) {
    await addQuestionRule(adminToken, examEId, qIT2, 1)
    await addQuestionRule(adminToken, examEId, qIT3, 2)
    // Leave as DRAFT intentionally
    log(`IT Advanced Security Assessment (DRAFT): 2 questions added, left in draft`)
  }

  void examEId

  // ── 8. Assignments ─────────────────────────────────────────────────────────
  log('\n── Assignments ──')

  // All Engineering employees → IT Security Fundamentals (deadline: 30 days)
  for (const empId of [alice, bob, carol, dave]) {
    await assignExamToUser(adminToken, examAId, empId, 30)
  }

  // All employees → Workplace Safety (deadline: 14 days — creates urgency for analytics demo)
  for (const empId of [alice, bob, carol, dave, eva, frank, grace, henry, iris, jack, kate]) {
    await assignExamToUser(adminToken, examBId, empId, 14)
  }

  // HR employees → HR Compliance Quiz (past deadline -7 days — for overdue section)
  for (const empId of [eva, frank, grace]) {
    await assignExamToUser(adminToken, examCId, empId, -7)
  }

  // All employees → Loyalty Survey (no deadline)
  for (const empId of [alice, bob, carol, dave, eva, frank, grace, henry, iris, jack, kate]) {
    await assignExamToUser(adminToken, examDId, empId, undefined)
  }

  log('All assignments created')

  // ── 9. Sessions — complete some exams ─────────────────────────────────────
  log('\n── Sessions ──')

  interface EmpSession { email: string; id: string }
  const empSessions: EmpSession[] = [
    { email: 'alice@bilimbaga-test.local', id: alice },
    { email: 'bob@bilimbaga-test.local', id: bob },
    { email: 'carol@bilimbaga-test.local', id: carol },
    { email: 'eva@bilimbaga-test.local', id: eva },
    { email: 'frank@bilimbaga-test.local', id: frank },
    { email: 'henry@bilimbaga-test.local', id: henry },
    { email: 'jack@bilimbaga-test.local', id: jack },
    { email: 'kate@bilimbaga-test.local', id: kate },
  ]

  for (const emp of empSessions) {
    const empLogin = await loginUser(emp.email, EMPLOYEE_PASS)
    if (!empLogin) { warn(`Could not login as ${emp.email} — skipping sessions`); continue }
    const empToken = empLogin.token

    // IT Security: Alice, Bob, Carol pass; Eva, Frank fail
    if ([alice, bob, carol].includes(emp.id)) {
      await completeSingleSession(empToken, examAId, true)
    } else if ([eva, frank].includes(emp.id)) {
      // Eva and Frank aren't assigned to IT Security, skip
    }

    // Loyalty Survey: everyone completes
    await completeSingleSession(empToken, examDId, true)

    // Safety exam: Jack and Kate submit with short text for manual grading
    // (all others skip for now to leave the grading queue populated)
    if ([jack, kate].includes(emp.id)) {
      await completeSingleSession(empToken, examBId, true)
    }
  }

  // HR Compliance: Eva and Frank complete (one pass, one fail), Grace doesn't start
  const evaLogin = await loginUser('eva@bilimbaga-test.local', EMPLOYEE_PASS)
  if (evaLogin) await completeSingleSession(evaLogin.token, examCId, true)
  const frankLogin = await loginUser('frank@bilimbaga-test.local', EMPLOYEE_PASS)
  if (frankLogin) await completeSingleSession(frankLogin.token, examCId, false)

  // Safety: Alice starts but doesn't submit (in-progress session for demo)
  const aliceLogin = await loginUser('alice@bilimbaga-test.local', EMPLOYEE_PASS)
  if (aliceLogin) {
    const startR = await apiPost<SessionCreated>(`${BASE}/api/v1/portal/exams/${examBId}/sessions`, {}, aliceLogin.token)
    if (startR.ok) log(`Alice has an in-progress Safety session: ${startR.data?.session_id}`)
  }

  // ── 10. Summary ────────────────────────────────────────────────────────────
  log('\n=== Seed Complete ===')
  log('Accounts:')
  log(`  Admin:        ${ADMIN_EMAIL} / ${ADMIN_KNOWN_PASS}`)
  log(`  Dept Admin:   da.engineering@bilimbaga-test.local / ${EMPLOYEE_PASS}`)
  log(`  Dept Admin:   da.hr@bilimbaga-test.local / ${EMPLOYEE_PASS}`)
  log(`  Examiner:     examiner@bilimbaga-test.local / ${EMPLOYEE_PASS}`)
  log(`  Employees:    alice..kate@bilimbaga-test.local / ${EMPLOYEE_PASS}`)
  log('')
  log('Exams:')
  log(`  IT Security Fundamentals (published, cert) → assigned to Engineering`)
  log(`  Workplace Safety Certification (published, cert, short text) → all employees`)
  log(`  HR Compliance Quiz (published) → HR dept (overdue deadline)`)
  log(`  Employee Loyalty & Values Survey (published, Likert) → all employees`)
  log(`  IT Advanced Security Assessment (DRAFT) → for authoring demo`)
  log('')
  log('Sessions created:')
  log(`  Alice, Bob, Carol: IT Security — PASSED`)
  log(`  Eva: HR Compliance — PASSED`)
  log(`  Frank: HR Compliance — FAILED`)
  log(`  Jack, Kate: Safety Cert — SUBMITTED (incl. short text awaiting grading)`)
  log(`  Everyone: Loyalty Survey — COMPLETED`)
  log(`  Alice: Safety Cert — IN PROGRESS (started but not submitted)`)
  log('')
  log(`Visit: ${BASE}`)
}

main().catch(err => {
  console.error('[seed] Fatal error:', err)
  process.exit(1)
})
