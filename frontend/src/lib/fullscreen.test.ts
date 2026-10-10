import { describe, it, expect, vi, afterEach } from 'vitest'
import { requestDocumentFullscreen, exitDocumentFullscreen } from './fullscreen'

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

  it('resolves once the request settles, even when it is refused', async () => {
    const request = vi.fn(() => Promise.reject(new Error('denied')))
    Object.defineProperty(document.documentElement, 'requestFullscreen', { configurable: true, value: request })
    await expect(requestDocumentFullscreen()).resolves.toBeUndefined()
  })
})

describe('exitDocumentFullscreen (#417)', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    Reflect.deleteProperty(document, 'exitFullscreen')
    Reflect.deleteProperty(document, 'fullscreenElement')
  })

  function setFullscreen(element: Element | null, exit: () => Promise<void>) {
    Object.defineProperty(document, 'fullscreenElement', { configurable: true, get: () => element })
    Object.defineProperty(document, 'exitFullscreen', { configurable: true, value: exit })
  }

  it('exits fullscreen when the document is fullscreen', async () => {
    const exit = vi.fn(() => Promise.resolve())
    setFullscreen(document.documentElement, exit)
    await exitDocumentFullscreen()
    expect(exit).toHaveBeenCalledTimes(1)
  })

  it('does nothing when the document is not fullscreen', async () => {
    const exit = vi.fn(() => Promise.resolve())
    setFullscreen(null, exit)
    await exitDocumentFullscreen()
    expect(exit).not.toHaveBeenCalled()
  })

  it('ignores a rejected exit request', async () => {
    const exit = vi.fn(() => Promise.reject(new Error('denied')))
    setFullscreen(document.documentElement, exit)
    await expect(exitDocumentFullscreen()).resolves.toBeUndefined()
  })

  it('does nothing when the API is missing', async () => {
    await expect(exitDocumentFullscreen()).resolves.toBeUndefined()
  })
})
