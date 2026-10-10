/**
 * ISS-165 live E2E: a department_admin only sees reports data of its own department subtree.
 *
 * Fixture: department A with a child A1, a separate department B, a department_admin of A,
 * one employee in each of A, A1 and B, and one exam assigned to all three employees, each of
 * whom submits a session. super_admin behaviour must be unchanged.
 *
 * Requires: make dev running; admin token from global-setup.ts. API-only (no browser).
 */
import { test, expect } from '@playwright/test'
import { requireTarget } from '../../scripts/lib/target-guard'
import { getSeedData, createTestQuestion, createTestExam, deleteTestExam, deleteTestQuestion } from './fixtures/seed'

const API = requireTarget('E2E_API_URL', process.env.E2E_API_URL || `http://localhost:${process.env.BB_API_PORT || 8080}`, process.env)
const NEW_PASSWORD = 'DeptScope1234!'

interface Envelope<T> {
  status: number
  data: T
  error: { code: string; message: string } | null
  text: string
}

async function call<T = any>(method: string, path: string, token?: string, body?: unknown): Promise<Envelope<T>> {
  const res = await fetch(`${API}${path}`, {
    method,
    headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  const text = await res.text()
  let json: { data?: T; error?: { code: string; message: string } | null } = {}
  try {
    json = JSON.parse(text)
  } catch {
    /* CSV bodies are not JSON */
  }
  return { status: res.status, data: json.data as T, error: json.error ?? null, text }
}

interface Actor {
  id: string
  email: string
  token: string
}

let adminToken = ''
let roleIds: Record<string, string> = {}
const departmentIds: string[] = []
const userIds: string[] = []
let questionId = ''
let examId = ''
let deptAdmin: Actor
let empA: Actor
let empA1: Actor
let empB: Actor

async function createDepartment(name: string, parentId: string | null): Promise<string> {
  const res = await call('POST', '/api/v1/departments', adminToken, { name, parent_id: parentId })
  expect(res.status, res.text).toBe(201)
  departmentIds.push(res.data.id)
  return res.data.id
}

async function createActor(prefix: string, role: string, departmentId: string): Promise<Actor> {
  const email = `${prefix}-${Date.now()}@e2e-test.local`
  const created = await call('POST', '/api/v1/users', adminToken, {
    email,
    full_name: `Scope ${prefix}`,
    role_id: roleIds[role],
    department_id: departmentId,
  })
  expect(created.status, created.text).toBe(201)
  userIds.push(created.data.id)
  const temp = created.data.temporary_password as string
  const first = await call('POST', '/api/v1/auth/login', undefined, { email, password: temp })
  expect(first.status, first.text).toBe(200)
  const changed = await call('POST', '/api/v1/auth/change-password', first.data.access_token, {
    current_password: temp,
    new_password: NEW_PASSWORD,
  })
  expect(changed.status, changed.text).toBe(200)
  const again = await call('POST', '/api/v1/auth/login', undefined, { email, password: NEW_PASSWORD })
  expect(again.status, again.text).toBe(200)
  return { id: created.data.id, email, token: again.data.access_token }
}

/** Assign the exam to the employee and let them submit a session (answering "Option A"). */
async function submitSessionFor(actor: Actor): Promise<void> {
  const assigned = await call('POST', `/api/v1/exams/${examId}/assign`, adminToken, {
    assignee_type: 'user',
    assignee_id: actor.id,
    deadline: null,
  })
  expect(assigned.status, assigned.text).toBeLessThan(300)
  const started = await call('POST', `/api/v1/portal/exams/${examId}/sessions`, actor.token, {})
  expect(started.status, started.text).toBeLessThan(300)
  const sessionId = started.data.session_id as string
  for (const q of started.data.questions as Array<{ id: string; options: Array<{ id: string; text: string }> }>) {
    const correct = q.options.find((o) => o.text === 'Option A')!
    const saved = await call('PUT', `/api/v1/portal/sessions/${sessionId}/answers/${q.id}`, actor.token, {
      selected_option_ids: [correct.id],
      text_answer: null,
      time_spent_seconds: 1,
    })
    expect(saved.status, saved.text).toBe(200)
  }
  const submitted = await call('POST', `/api/v1/portal/sessions/${sessionId}/submit`, actor.token, {})
  expect(submitted.status, submitted.text).toBeLessThan(300)
}

test.describe.configure({ mode: 'serial' })

test.beforeAll(async () => {
  adminToken = (await getSeedData()).adminToken
  const roles = await call<Array<{ id: string; name: string }>>('GET', '/api/v1/users/roles', adminToken)
  roleIds = Object.fromEntries(roles.data.map((r) => [r.name, r.id]))

  const stamp = Date.now()
  const a = await createDepartment(`E2E Scope A ${stamp}`, null)
  const a1 = await createDepartment(`E2E Scope A1 ${stamp}`, a)
  const b = await createDepartment(`E2E Scope B ${stamp}`, null)

  deptAdmin = await createActor('scope-admin', 'department_admin', a)
  empA = await createActor('scope-emp-a', 'employee', a)
  empA1 = await createActor('scope-emp-a1', 'employee', a1)
  empB = await createActor('scope-emp-b', 'employee', b)

  const question = await createTestQuestion(adminToken, `Scope question ${stamp}`, 'single', true)
  questionId = question.id
  const exam = await createTestExam(adminToken, `E2E Scope Exam ${stamp}`, questionId)
  examId = exam.id
  for (const actor of [empA, empA1, empB]) await submitSessionFor(actor)
})

test.afterAll(async () => {
  if (examId) await deleteTestExam(adminToken, examId)
  if (questionId) await deleteTestQuestion(adminToken, questionId)
  for (const id of userIds) await call('DELETE', `/api/v1/users/${id}`, adminToken)
  for (const id of departmentIds.slice().reverse()) await call('DELETE', `/api/v1/departments/${id}`, adminToken)
})

test.describe('department_admin scoping (ISS-165)', () => {
  test('employee record, progress and CSV: own and descendant department OK, other department 404', async () => {
    for (const path of ['record', 'progress', 'record/export']) {
      for (const target of [empA, empA1]) {
        const ok = await call('GET', `/api/v1/admin/users/${target.id}/${path}`, deptAdmin.token)
        expect(ok.status, `${path} ${target.email}`).toBe(200)
      }
      const denied = await call('GET', `/api/v1/admin/users/${empB.id}/${path}`, deptAdmin.token)
      expect(denied.status, `${path} other department`).toBe(404)
      expect(denied.error?.code).toBe('USER_NOT_FOUND')
      expect(denied.text).not.toContain(empB.email)
      // Indistinguishable from an unknown id (no existence leak).
      const unknown = await call('GET', `/api/v1/admin/users/00000000-0000-4000-8000-000000000000/${path}`, deptAdmin.token)
      expect(unknown.status).toBe(404)
      expect(denied.text).toBe(unknown.text)
    }
    // GET /users/{id} for another department is 404 NOT_FOUND, the same body as an unknown id (FR-BB117 D-4c / D-5).
    const viaUsers = await call('GET', `/api/v1/users/${empB.id}`, deptAdmin.token)
    expect(viaUsers.status).toBe(404)
    expect(viaUsers.error?.code).toBe('NOT_FOUND')
    const unknownUser = await call('GET', '/api/v1/users/00000000-0000-4000-8000-000000000000', deptAdmin.token)
    expect(viaUsers.text).toBe(unknownUser.text)

    // An employee reading another user is refused: a caller without users:read gets 403.
    const employeeViaUsers = await call('GET', `/api/v1/users/${empB.id}`, empA.token)
    expect(employeeViaUsers.status).toBe(403)

    // super_admin unchanged.
    const sa = await call('GET', `/api/v1/admin/users/${empB.id}/record`, adminToken)
    expect(sa.status).toBe(200)
  })

  test('exam analytics aggregate only the department subtree', async () => {
    const scoped = await call('GET', `/api/v1/admin/exams/${examId}/analytics`, deptAdmin.token)
    expect(scoped.status, scoped.text).toBe(200)
    expect(scoped.data.total_attempts).toBe(2)
    expect(scoped.data.unique_participants).toBe(2)

    const all = await call('GET', `/api/v1/admin/exams/${examId}/analytics`, adminToken)
    expect(all.data.total_attempts).toBe(3)
  })

  test('exam results CSV lists only the department subtree', async () => {
    const scoped = await call('GET', `/api/v1/admin/exams/${examId}/results/export`, deptAdmin.token)
    expect(scoped.status).toBe(200)
    expect(scoped.text).toContain('Scope scope-emp-a')
    expect(scoped.text).toContain('Scope scope-emp-a1')
    expect(scoped.text).not.toContain('Scope scope-emp-b')

    const all = await call('GET', `/api/v1/admin/exams/${examId}/results/export`, adminToken)
    expect(all.text).toContain('Scope scope-emp-b')
  })

  test('dashboard lists and aggregates only the department subtree', async () => {
    const scoped = await call('GET', '/api/v1/admin/dashboard', deptAdmin.token)
    expect(scoped.status, scoped.text).toBe(200)
    const names = (scoped.data.recent_activity as Array<{ employee_name: string }>).map((r) => r.employee_name)
    expect(names).toContain('Scope scope-emp-a')
    expect(names).toContain('Scope scope-emp-a1')
    expect(names).not.toContain('Scope scope-emp-b')
    const row = (scoped.data.completion_rate_by_exam as Array<{ exam_id: string; assigned_count: number }>).find(
      (r) => r.exam_id === examId,
    )
    expect(row?.assigned_count).toBe(2)
    // Exams with no participant in the subtree are not listed for a department_admin.
    for (const r of scoped.data.completion_rate_by_exam as Array<{ assigned_count: number }>) {
      expect(r.assigned_count).toBeGreaterThan(0)
    }

    const all = await call('GET', '/api/v1/admin/dashboard', adminToken)
    const allNames = (all.data.recent_activity as Array<{ employee_name: string }>).map((r) => r.employee_name)
    expect(allNames).toContain('Scope scope-emp-b')
    const allRow = (all.data.completion_rate_by_exam as Array<{ exam_id: string; assigned_count: number }>).find(
      (r) => r.exam_id === examId,
    )
    expect(allRow?.assigned_count).toBe(3)
  })

  test('dashboard PDF export still succeeds for a department_admin', async () => {
    const res = await fetch(`${API}/api/v1/admin/dashboard/export`, { headers: { Authorization: `Bearer ${deptAdmin.token}` } })
    expect(res.status).toBe(200)
    expect(res.headers.get('content-type')).toContain('application/pdf')
  })
})
