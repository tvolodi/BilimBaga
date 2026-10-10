import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { afterEach, describe, expect, it, vi } from 'vitest'

// FR-BB321 AC-3: the first-paint script is a plain file, read here and run as written.
// Vitest runs from frontend/, so the paths are relative to it.
const scriptText = readFileSync(resolve(process.cwd(), 'public', 'theme-init.js'), 'utf8')
const indexHtml = readFileSync(resolve(process.cwd(), 'index.html'), 'utf8')

function stubPrefersDark(prefersDark: boolean) {
  window.matchMedia = vi.fn(
    (query: string) => ({ matches: prefersDark, media: query }) as unknown as MediaQueryList,
  )
}

const SWITCH_LINE = 'var THEME_SWITCH_ENABLED = false;'

// The shipped file has the switch off. The enabled-state tests flip the constant in the text
// they run; the off-state tests run the file as shipped.
function scriptWithSwitch(enabled: boolean): string {
  if (!scriptText.includes(SWITCH_LINE)) throw new Error('switch line not found in theme-init.js')
  return scriptText.replace(SWITCH_LINE, `var THEME_SWITCH_ENABLED = ${enabled};`)
}

function runThemeInit(enabled = true) {
  new Function(scriptWithSwitch(enabled))()
}

function htmlIsDark() {
  return document.documentElement.classList.contains('dark')
}

describe('theme-init.js while the switch is off (FR-BB321 part 1)', () => {
  it('ships with the switch off, matching ThemeProvider', () => {
    expect(scriptText).toContain(SWITCH_LINE)
  })

  it('stays light for a stored dark value', () => {
    stubPrefersDark(false)
    window.localStorage.setItem('bb-theme', 'dark')
    runThemeInit(false)
    expect(htmlIsDark()).toBe(false)
    expect(document.documentElement.style.colorScheme).toBe('light')
  })

  it('stays light when the OS prefers dark and nothing is stored', () => {
    stubPrefersDark(true)
    runThemeInit(false)
    expect(htmlIsDark()).toBe(false)
    expect(document.documentElement.style.colorScheme).toBe('light')
  })

  it('removes a dark class left on the page', () => {
    stubPrefersDark(false)
    document.documentElement.classList.add('dark')
    runThemeInit(false)
    expect(htmlIsDark()).toBe(false)
  })
})

afterEach(() => {
  delete (window as { matchMedia?: unknown }).matchMedia
  window.localStorage.clear()
  document.documentElement.classList.remove('dark')
  document.documentElement.style.colorScheme = ''
  vi.restoreAllMocks()
})

describe('theme-init.js', () => {
  it('applies a stored dark value', () => {
    stubPrefersDark(false)
    window.localStorage.setItem('bb-theme', 'dark')
    runThemeInit()
    expect(htmlIsDark()).toBe(true)
    expect(document.documentElement.style.colorScheme).toBe('dark')
  })

  it('applies a stored light value even when the OS is dark', () => {
    stubPrefersDark(true)
    window.localStorage.setItem('bb-theme', 'light')
    runThemeInit()
    expect(htmlIsDark()).toBe(false)
    expect(document.documentElement.style.colorScheme).toBe('light')
  })

  it('treats an invalid stored value as system', () => {
    stubPrefersDark(true)
    window.localStorage.setItem('bb-theme', 'purple')
    runThemeInit()
    expect(htmlIsDark()).toBe(true)
    expect(document.documentElement.style.colorScheme).toBe('dark')
  })

  it('treats a throwing storage as system', () => {
    stubPrefersDark(true)
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('storage blocked')
    })
    expect(() => runThemeInit()).not.toThrow()
    expect(htmlIsDark()).toBe(true)
    expect(document.documentElement.style.colorScheme).toBe('dark')
  })

  it('follows a dark OS when nothing is stored', () => {
    stubPrefersDark(true)
    runThemeInit()
    expect(htmlIsDark()).toBe(true)
    expect(document.documentElement.style.colorScheme).toBe('dark')
  })

  it('follows a light OS when nothing is stored', () => {
    stubPrefersDark(false)
    runThemeInit()
    expect(htmlIsDark()).toBe(false)
    expect(document.documentElement.style.colorScheme).toBe('light')
  })

  it('does not throw and keeps the light default when matchMedia is missing', () => {
    expect(() => runThemeInit()).not.toThrow()
    expect(htmlIsDark()).toBe(false)
  })

  it('is a classic script of at most 30 lines', () => {
    expect(scriptText.split('\n').length).toBeLessThanOrEqual(30)
    expect(scriptText).not.toMatch(/\bimport\s|\bexport\s/)
  })
})

describe('index.html', () => {
  it('loads theme-init.js as a plain script in the head, before the app module', () => {
    const head = indexHtml.slice(indexHtml.indexOf('<head>'), indexHtml.indexOf('</head>'))
    const tag = '<script src="/theme-init.js"></script>'
    expect(head).toContain(tag)
    expect(indexHtml.indexOf(tag)).toBeLessThan(indexHtml.indexOf('<script type="module"'))
    expect(indexHtml.indexOf(tag)).toBeLessThan(indexHtml.indexOf('</head>'))
  })
})
