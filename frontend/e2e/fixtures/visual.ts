/**
 * Shared helpers of the visual regression suite (FR-BB320 AC-13). The fixtures are created once per run by the setup
 * project (visual-fixtures.setup.ts) and written to .auth/visual-fixtures.json; the page tests only read that file.
 * Nothing here logs in: a login per worker hits the auth rate limiter.
 */
import type { Page } from '@playwright/test'
import fs from 'fs'
import path from 'path'
import { fileURLToPath } from 'url'

const AUTH_DIR = path.join(path.dirname(fileURLToPath(import.meta.url)), '..', '..', '.auth')
export const ADMIN_STORAGE_STATE = path.join(AUTH_DIR, 'admin.json')
export const EMPLOYEE_STORAGE_STATE = path.join(AUTH_DIR, 'employee.json')
/** The employee token that global setup saved. */
const EMPLOYEE_TOKEN_FILE = path.join(AUTH_DIR, 'employee-token.txt')
export const VISUAL_FIXTURES_FILE = path.join(AUTH_DIR, 'visual-fixtures.json')
export const API = process.env.E2E_API_URL || 'http://localhost:8080'

export interface VisualFixtures {
  takingSessionId: string
  resultSessionId: string
  questionId: string
  examIds: string[]
}

/**
 * Loads every Inter weight the app uses before a capture. The font stylesheet loads from Google Fonts after the first
 * paint, so without this a badge can be drawn in a fallback face in one run and in Inter in the next (FR-BB320 AC-13).
 */
export async function loadAppFonts(page: Page): Promise<void> {
  await page.evaluate(async () => {
    await Promise.all(['300', '400', '500', '600', '700'].map((weight) => document.fonts.load(`${weight} 16px Inter`)))
    await document.fonts.ready
  })
}

export function readVisualFixtures(): VisualFixtures {
  return JSON.parse(fs.readFileSync(VISUAL_FIXTURES_FILE, 'utf8')) as VisualFixtures
}

/** Authenticated call with the saved employee token. */
export function employeeFetch(url: string, method: string, body?: unknown): Promise<Response> {
  return fetch(`${API}${url}`, {
    method,
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${fs.readFileSync(EMPLOYEE_TOKEN_FILE, 'utf8').trim()}`,
    },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
}

interface StartedSession {
  session_id: string
  questions: Array<{ id: string; options: Array<{ id: string; text: string }> }>
}

/** Opens a session on the exam for the seeded employee. */
export async function startSession(examId: string): Promise<string> {
  const res = await employeeFetch(`/api/v1/portal/exams/${examId}/sessions`, 'POST', {})
  const body = (await res.json()) as { data: StartedSession | null }
  if (!res.ok || !body.data?.session_id) throw new Error(`start session failed: ${res.status}`)
  return body.data.session_id
}

/** A submitted, passed session: every question answered with the option labelled "Option A" (the fixture's correct one). */
export async function passedSession(examId: string): Promise<string> {
  const res = await employeeFetch(`/api/v1/portal/exams/${examId}/sessions`, 'POST', {})
  const body = (await res.json()) as { data: StartedSession | null }
  if (!res.ok || !body.data?.session_id) throw new Error(`start session failed: ${res.status}`)
  const { session_id: sessionId, questions } = body.data
  for (const q of questions) {
    const correct = q.options.find((o) => o.text === 'Option A')
    if (!correct) throw new Error(`question ${q.id} has no "Option A" option`)
    const answer = await employeeFetch(`/api/v1/portal/sessions/${sessionId}/answers/${q.id}`, 'PUT', {
      selected_option_ids: [correct.id],
      text_answer: null,
      time_spent_seconds: 1,
    })
    if (!answer.ok) throw new Error(`saving an answer failed: ${answer.status}`)
  }
  const submit = await employeeFetch(`/api/v1/portal/sessions/${sessionId}/submit`, 'POST', {})
  if (!submit.ok) throw new Error(`submit failed: ${submit.status}`)
  return sessionId
}
