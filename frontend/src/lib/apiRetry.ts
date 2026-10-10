/** The app's default retry count for queries (see createAppQueryClient). */
const MAX_RETRIES = 3

/**
 * True for a not-found answer: an HTTP 404, or a not-found code. The shared apiFetch error carries the HTTP
 * status as `status`; the exam layer's ExamApiError carries it as `httpStatus` (#487). The API answers an
 * unknown exam with ERR_NOT_FOUND, so that code counts too (EXAM_NOT_FOUND is kept for older answers).
 */
export function isNotFoundError(error: unknown): boolean {
  const e = error as { status?: unknown; httpStatus?: unknown; code?: unknown } | null
  return (
    e?.status === 404 ||
    e?.httpStatus === 404 ||
    e?.code === 'ERR_NOT_FOUND' ||
    e?.code === 'EXAM_NOT_FOUND'
  )
}

/**
 * Retry policy for queries that load one resource by id (#470). A not-found answer does not change
 * on a retry, so it is not retried: each retry only delays the not-found state behind a loading skeleton.
 */
export function retryUnlessNotFound(failureCount: number, error: unknown): boolean {
  return !isNotFoundError(error) && failureCount < MAX_RETRIES
}
