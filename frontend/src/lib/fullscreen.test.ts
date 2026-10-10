import { describe, it, expect, vi, afterEach } from 'vitest'
import { requestDocumentFullscreen } from './fullscreen'

describe('requestDocumentFullscreen', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    Reflect.deleteProperty(document.documentElement, 'requestFullscreen')
  })

  it('calls requestFullscreen on the document element', () => {
    const request = vi.fn(() => Promise.resolve())
    Object.defineProperty(document.documentElement, 'requestFullscreen', { configurable: true, value: request })
    requestDocumentFullscreen()
    expect(request).toHaveBeenCalledTimes(1)
  })

  it('ignores a rejected request without an unhandled rejection', async () => {
    const request = vi.fn(() => Promise.reject(new Error('denied')))
    Object.defineProperty(document.documentElement, 'requestFullscreen', { configurable: true, value: request })
    const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => undefined)
    expect(() => requestDocumentFullscreen()).not.toThrow()
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(errorSpy).not.toHaveBeenCalled()
  })

  it('does nothing when the API is missing', () => {
    expect(() => requestDocumentFullscreen()).not.toThrow()
  })

  it('ignores a synchronous throw', () => {
    const request = vi.fn(() => {
      throw new Error('not allowed')
    })
    Object.defineProperty(document.documentElement, 'requestFullscreen', { configurable: true, value: request })
    expect(() => requestDocumentFullscreen()).not.toThrow()
  })
})
