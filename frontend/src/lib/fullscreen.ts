/**
 * Asks the browser to show the whole document fullscreen (FR-BB319 AC-8, D-3).
 *
 * Must be called synchronously from a user gesture (a click handler). Failure is silent by design:
 * an unsupported browser (for example iOS Safari) or a refused request leaves the exam running
 * without fullscreen, and no error is surfaced to the candidate.
 *
 * The returned promise always resolves, so callers can await it before acting (for example, before
 * leaving fullscreen again when the session could not be created).
 */
export function requestDocumentFullscreen(): Promise<void> {
  try {
    return document.documentElement.requestFullscreen().catch(() => undefined)
  } catch {
    // requestFullscreen is missing or threw synchronously: ignore.
    return Promise.resolve()
  }
}

/**
 * Leaves fullscreen if the document is currently fullscreen (#417). Used when the exam could not be
 * started, so a failed start does not leave the candidate stuck in fullscreen on the portal page.
 * A no-op when nothing is fullscreen; failures are ignored, as in requestDocumentFullscreen.
 */
export function exitDocumentFullscreen(): Promise<void> {
  try {
    if (!document.fullscreenElement) return Promise.resolve()
    return document.exitFullscreen().catch(() => undefined)
  } catch {
    return Promise.resolve()
  }
}
