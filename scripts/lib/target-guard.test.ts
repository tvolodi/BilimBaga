import { describe, it, expect } from 'vitest'
import { checkTarget, requireTarget } from './target-guard'

const none = {}
const allow = { ALLOW_PROTECTED_HOST: '1' }
const ok = (u: string, env = none) => checkTarget('T', u, env).ok

describe('target-guard allowlist', () => {
  it.each([
    'http://localhost:8080',
    'http://localhost',
    'https://LOCALHOST:5173/',
    'http://127.0.0.1:18080',
    'http://[::1]:8080',
    'http://app.localhost:3000',
    'http://localhost.:8080',
    'https://bilimbaga-qa.ai-dala.com',
    'https://BILIMBAGA-QA.ai-dala.com./api',
  ])('allows %s without a flag', (u) => {
    expect(ok(u)).toBe(true)
  })

  it.each([
    ['plain', 'https://bilimbaga-test.ai-dala.com'],
    ['http', 'http://bilimbaga-test.ai-dala.com'],
    ['uppercase', 'https://BILIMBAGA-TEST.AI-DALA.COM'],
    ['trailing dot', 'https://bilimbaga-test.ai-dala.com.'],
    ['port', 'https://bilimbaga-test.ai-dala.com:8443'],
    ['userinfo', 'https://user:pw@bilimbaga-test.ai-dala.com'],
    ['userinfo trick', 'https://localhost@bilimbaga-test.ai-dala.com'],
    ['percent-encoded hyphen', 'https://bilimbaga%2Dtest.ai-dala.com'],
    ['percent-encoded dot', 'https://bilimbaga-test%2Eai-dala.com'],
    ['ideographic dot', 'https://bilimbaga-test。ai-dala。com'],
    ['fullwidth letters', 'https://ｂilimbaga-test.ai-dala.com'],
    ['subdomain', 'https://x.bilimbaga-test.ai-dala.com'],
    ['path', 'https://bilimbaga-test.ai-dala.com/api/v1'],
  ])('refuses protected host (%s)', (_n, u) => {
    const d = checkTarget('T', u, none)
    expect(d.ok).toBe(false)
    expect(d.reason).toMatch(/protected/)
  })

  it.each([
    'https://example.com',
    'http://10.0.0.5:8080',
    'https://localhost.evil.com',
    'https://evilbilimbaga-qa.ai-dala.com',
    'https://bilimbaga.ai-dala.com',
  ])('refuses non-allowlisted host %s', (u) => {
    expect(ok(u)).toBe(false)
  })

  it('refuses invalid URLs, empty values and non-http schemes', () => {
    expect(checkTarget('T', 'not a url', none).ok).toBe(false)
    expect(checkTarget('T', 'localhost:8080', none).ok).toBe(false)
    expect(checkTarget('T', '', none).ok).toBe(false)
    expect(checkTarget('T', undefined, none).ok).toBe(false)
    expect(checkTarget('T', 'ftp://localhost', none).ok).toBe(false)
    expect(checkTarget('T', 'file:///etc/passwd', none).ok).toBe(false)
  })

  it('escape hatch allows other hosts; only exactly "1" counts', () => {
    expect(ok('https://example.com', allow)).toBe(true)
    expect(ok('https://example.com', { ALLOW_PROTECTED_HOST: 'true' })).toBe(false)
    expect(ok('https://example.com', { ALLOW_PROTECTED_HOST: '0' })).toBe(false)
  })

  it('protected host with the flag is allowed but carries a loud warning, bypass spellings included', () => {
    for (const u of [
      'https://bilimbaga-test.ai-dala.com',
      'https://bilimbaga%2Dtest.ai-dala.com',
      'https://bilimbaga-test。ai-dala。com',
    ]) {
      const d = checkTarget('T', u, allow)
      expect(d.ok).toBe(true)
      expect(d.warning).toMatch(/PRODUCTION-CLASS/)
    }
    expect(checkTarget('T', 'http://localhost:8080', allow).warning).toBeUndefined()
  })

  it('returns a normalised URL without trailing slash', () => {
    expect(checkTarget('T', 'http://LocalHost:5173/', none).url).toBe('http://localhost:5173')
  })

  it('requireTarget throws on refusal and returns the URL otherwise', () => {
    expect(() => requireTarget('E2E_API_URL', 'https://bilimbaga%2Dtest.ai-dala.com', none)).toThrow(/E2E_API_URL/)
    expect(requireTarget('E2E_BASE_URL', 'http://localhost:5173', none)).toBe('http://localhost:5173')
  })
})
