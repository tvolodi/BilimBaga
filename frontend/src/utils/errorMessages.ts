export const ERROR_CODE_MAP: Record<string, string> = {
  // Auth / token
  MISSING_TOKEN: 'errors.missingToken',
  INVALID_TOKEN: 'errors.invalidToken',
  TOKEN_EXPIRED: 'errors.tokenExpired',
  UNAUTHORIZED: 'errors.unauthorized',
  FORBIDDEN: 'errors.forbidden',

  // General
  ERR_INTERNAL: 'errors.internal',
  INTERNAL_ERROR: 'errors.internal',
  ERR_INVALID_BODY: 'errors.invalidBody',
  INVALID_BODY: 'errors.invalidBody',
  ERR_NOT_FOUND: 'errors.notFound',
  NOT_FOUND: 'errors.notFound',
  VALIDATION_ERROR: 'errors.validation',
  ERR_VALIDATION: 'errors.validation',
  ERR_INVALID_PARAM: 'errors.invalidParam',

  // Exams
  ERR_INVALID_TRANSITION: 'errors.invalidTransition',
  ERR_NOT_DRAFT: 'errors.examNotDraft',
  EXAM_NOT_DRAFT: 'errors.examNotDraft',
  EXAM_NOT_ACTIVE: 'errors.examNotActive',
  EXAM_ARCHIVED: 'errors.examArchived',
  EXAM_NOT_FOUND: 'errors.examNotFound',
  EXAM_NOT_CERTIFIABLE: 'errors.examNotCertifiable',
  EXAM_OUTSIDE_WINDOW: 'errors.examOutsideWindow',
  ASSIGNMENT_ALREADY_EXISTS: 'errors.assignmentAlreadyExists',
  ERR_DEADLINE_IN_PAST: 'errors.deadlineInPast',
  RULE_NOT_MANUAL: 'errors.ruleNotManual',

  // Sessions
  SESSION_NOT_FOUND: 'errors.sessionNotFound',
  SESSION_FORBIDDEN: 'errors.sessionForbidden',
  SESSION_ALREADY_OPEN: 'errors.sessionAlreadyOpen',
  SESSION_NOT_ACTIVE: 'errors.sessionNotActive',
  SESSION_EXPIRED: 'errors.sessionExpired',
  SESSION_NOT_SUBMITTED: 'errors.sessionNotSubmitted',
  SESSION_NOT_PASSED: 'errors.sessionNotPassed',
  SESSION_IN_PROGRESS: 'errors.sessionInProgress',
  ATTEMPTS_EXHAUSTED: 'errors.attemptsExhausted',
  INSUFFICIENT_QUESTIONS: 'errors.insufficientQuestions',
  EXAM_NOT_ASSIGNED: 'errors.examNotAssigned',
  INVALID_TIME_SPENT: 'errors.invalidTimeSpent',
  INVALID_OPTION: 'errors.invalidOption',
  INVALID_ANSWER_FORMAT: 'errors.invalidAnswerFormat',
  INVALID_EVENT_TYPE: 'errors.invalidEventType',
  INVALID_SCORE: 'errors.invalidScore',
  QUESTION_NOT_IN_SESSION: 'errors.questionNotInSession',
  INVALID_DATE: 'errors.invalidDate',

  // Questions
  ERR_STEM_REQUIRED: 'errors.stemRequired',

  // Categories
  ERR_PARENT_NOT_FOUND: 'errors.parentNotFound',
  ERR_CATEGORY_CYCLE: 'errors.categoryCycle',
  ERR_CATEGORY_IN_USE: 'errors.categoryInUse',
  ERR_INVALID_NAME: 'errors.invalidName',

  // Departments
  DUPLICATE_NAME: 'errors.duplicateName',
  DEPARTMENT_HAS_CHILDREN: 'errors.departmentHasChildren',
  DEPARTMENT_NOT_EMPTY: 'errors.departmentNotEmpty',

  // Users
  DUPLICATE_EMAIL: 'errors.duplicateEmail',
  MISSING_FILE: 'errors.missingFile',
  INVALID_CSV: 'errors.invalidCsv',
  TOO_MANY_ROWS: 'errors.tooManyRows',
  USER_NOT_FOUND: 'errors.userNotFound',

  // Email
  EMAIL_UNAVAILABLE: 'errors.emailUnavailable',
}

/**
 * Resolves a backend error code to an i18n key.
 * Returns a fallback key if the code is not in the map.
 */
export function resolveErrorKey(code: string | undefined): string {
  if (!code) return 'errors.internal'
  return ERROR_CODE_MAP[code] ?? 'errors.internal'
}
