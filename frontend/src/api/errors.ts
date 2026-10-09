/** Error carrying the API envelope's machine-readable code (e.g. PASSWORD_CHANGE_REQUIRED). */
export interface CodedError extends Error {
  code: string
}

/**
 * Build an Error from an API error envelope, preserving `code` so global handlers
 * (QueryCache/MutationCache onError, retry policy) can recognise it (ISS-160).
 */
export function errorWithCode(e: { code?: string; message: string }): CodedError {
  const err = new Error(e.message) as CodedError
  err.code = e.code ?? 'ERR_UNKNOWN'
  return err
}
