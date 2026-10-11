import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

// #511: the Google Fonts stylesheet must not block the first paint. It loads with media="print" and
// switches to "all" on load, with a noscript copy for clients without script. The onload handler is inline,
// so it needs script-src 'unsafe-inline' (deploy/nginx.conf). If that is ever removed, move the switch to
// theme-init.js instead of letting the font stay on media="print".
describe('index.html font stylesheet (#511)', () => {
  const html = readFileSync(resolve(process.cwd(), 'index.html'), 'utf8')
  const fontUrl = 'https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap'

  it('loads the font stylesheet without blocking the first paint', () => {
    // The stylesheet link is the one with the css2 path; the preconnect link has no such path.
    const link = /<link[^>]*css2\?family=[^>]*>/s.exec(html)?.[0] ?? ''
    expect(link).toContain('rel="stylesheet"')
    expect(link).toContain('media="print"')
    expect(link).toContain(`onload="this.media='all'"`)
  })

  it('keeps a noscript copy of the stylesheet', () => {
    const noscript = /<noscript>([\s\S]*?)<\/noscript>/.exec(html)?.[1] ?? ''
    expect(noscript).toContain(fontUrl)
  })
})
