import type { TFunction } from 'i18next'

/** Parsed, safe view of a publish-validation `details` payload. All fields optional. */
export interface InsufficientDetails {
  ruleId?: string
  difficulty?: string
  required?: number
  available?: number
}

function asRecord(v: unknown): Record<string, unknown> | null {
  return v !== null && typeof v === 'object' && !Array.isArray(v) ? (v as Record<string, unknown>) : null
}

function asNum(v: unknown): number | undefined {
  return typeof v === 'number' && Number.isFinite(v) ? v : undefined
}

function asStr(v: unknown): string | undefined {
  return typeof v === 'string' && v.trim() !== '' ? v : undefined
}

/**
 * Normalises the backend `details` (object for INSUFFICIENT_ADAPTIVE_QUESTIONS,
 * array of rules, string, or missing) into a flat, typed record. Never throws.
 * For an array the first entry that is an object is used.
 */
export function parseInsufficientDetails(details: unknown): InsufficientDetails {
  const rec = Array.isArray(details) ? asRecord(details.find((d) => asRecord(d))) : asRecord(details)
  if (!rec) return {}
  return {
    ruleId: asStr(rec.rule_id),
    difficulty: asStr(rec.difficulty),
    required: asNum(rec.required),
    available: asNum(rec.available),
  }
}

/** Always returns a plain string, never "[object Object]". */
export function safeMessage(message: unknown): string {
  return typeof message === 'string' ? message : ''
}

interface ErrorLike {
  code?: unknown
  message?: unknown
  details?: unknown
}

/**
 * Localised, human-readable text for an exam save/publish error.
 * Handles INSUFFICIENT_ADAPTIVE_QUESTIONS and INSUFFICIENT_QUESTIONS with
 * interpolated counts when available; falls back to a generic localised message.
 */
export function describeExamError(err: unknown, t: TFunction): string {
  const e = (err ?? {}) as ErrorLike
  const code = typeof e.code === 'string' ? e.code : ''

  if (code === 'INSUFFICIENT_QUESTIONS') {
    return t('exam.wizard.step4.noQuestionsError')
  }
  if (code === 'INSUFFICIENT_ADAPTIVE_QUESTIONS') {
    const d = parseInsufficientDetails(e.details)
    if (d.required !== undefined && d.available !== undefined) {
      const key = d.difficulty
        ? 'exam.wizard.step4.adaptiveInsufficientDetailed'
        : 'exam.wizard.step4.adaptiveInsufficientCounts'
      return t(key, {
        required: d.required,
        available: d.available,
        difficulty: d.difficulty ? t(`questionBank.difficulty.${d.difficulty}`, { defaultValue: d.difficulty }) : '',
        id: d.ruleId ? d.ruleId.slice(0, 8) : '',
      })
    }
    return t('exam.wizard.step4.adaptiveInsufficient')
  }
  return safeMessage(e.message) || t('exam.wizard.step4.genericError')
}
