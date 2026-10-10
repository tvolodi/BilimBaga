/**
 * Asks the browser to show the whole document fullscreen (FR-BB319 AC-8, D-3).
 *
 * Must be called synchronously from a user gesture (a click handler). Failure is silent by design:
 * an unsupported browser (for example iOS Safari) or a refused request leaves the exam running
 * without fullscreen, and no error is surfaced to the candidate.
 */
export function requestDocumentFullscreen(): void {
  try {
    void document.documentElement.requestFullscreen().catch(() => undefined)
  } catch {
    // requestFullscreen is missing or threw synchronously: ignore.
  }
}
