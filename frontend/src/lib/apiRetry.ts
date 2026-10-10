/** The app's default retry count for queries (see createAppQueryClient). */
const MAX_RETRIES = 3

/** True for a 404 or an EXAM_NOT_FOUND response. */
export function isNotFoundError(error: unknown): boolean {
  const e = error as { status?: unknown; code?: unknown } | null
  return e?.status === 404 || e?.code === 'EXAM_NOT_FOUND'
}

/**
 * Retry policy for queries that load one resource by id (#470). A not-found answer does not change
 * on a retry, so it is not retried: each retry only delays the not-found state behind a loading skeleton.
 */
export function retryUnlessNotFound(failureCount: number, error: unknown): boolean {
  return !isNotFoundError(error) && failureCount < MAX_RETRIES
}
