/**
 * Allowlist guard for test/seed targets (ISS-107, DEC-001).
 *
 * Scripts and e2e setup that create users, log in or write data must only hit local stacks or the QA
 * instance. bilimbaga-test.ai-dala.com is a customer demo (production-class). The decision is made on the
 * URL-normalised hostname (WHATWG URL: percent-escapes decoded, ideographic dots and fullwidth letters
 * mapped, lowercased), never on the raw string. Pure module: no process/env access (env is a parameter).
 */

export const PROTECTED_HOST = 'bilimbaga-test.ai-dala.com'
export const QA_HOST = 'bilimbaga-qa.ai-dala.com'

export interface TargetDecision {
  ok: boolean
  /** Normalised URL without trailing slash (set when ok). */
  url?: string
  hostname?: string
  /** Why the target was refused (set when !ok). */
  reason?: string
  /** Loud warning to print when allowed only via the escape hatch on the protected host. */
  warning?: string
}

function isAllowedHost(h: string): boolean {
  return (
    h === 'localhost' ||
    h.endsWith('.localhost') ||
    h === '127.0.0.1' ||
    h === '::1' ||
    h === QA_HOST
  )
}

function isProtectedHost(h: string): boolean {
  return h === PROTECTED_HOST || h.endsWith('.' + PROTECTED_HOST)
}

export function checkTarget(
  name: string,
  raw: string | undefined,
  env: { ALLOW_PROTECTED_HOST?: string },
): TargetDecision {
  if (!raw) return { ok: false, reason: `${name} is empty` }
  let u: URL
  try {
    u = new URL(raw)
  } catch {
    return { ok: false, reason: `${name} is not a valid URL: ${JSON.stringify(raw)}` }
  }
  if (u.protocol !== 'http:' && u.protocol !== 'https:') {
    return { ok: false, reason: `${name} must use http or https (got ${u.protocol})` }
  }
  let host = u.hostname.toLowerCase()
  if (host.startsWith('[') && host.endsWith(']')) host = host.slice(1, -1)
  host = host.replace(/\.+$/, '')
  const url = u.href.replace(/\/+$/, '')
  const flag = env.ALLOW_PROTECTED_HOST === '1'

  if (isProtectedHost(host)) {
    if (!flag) {
      return {
        ok: false,
        hostname: host,
        reason: `${name} resolves to ${host}: protected production-class host (customer demo). Set ALLOW_PROTECTED_HOST=1 only with explicit user approval.`,
      }
    }
    return {
      ok: true,
      url,
      hostname: host,
      warning: `WARNING: ${name} targets ${host}, a PRODUCTION-CLASS customer demo. ALLOW_PROTECTED_HOST=1 is set; this writes/logs in against live data.`,
    }
  }
  if (isAllowedHost(host)) return { ok: true, url, hostname: host }
  if (flag) return { ok: true, url, hostname: host }
  return {
    ok: false,
    hostname: host,
    reason: `${name} host ${host} is not on the allowlist (localhost, 127.0.0.1, ::1, *.localhost, ${QA_HOST}). Set ALLOW_PROTECTED_HOST=1 only with explicit user approval.`,
  }
}

/** Throws on refusal; logs the protected-host warning. Returns the normalised URL. */
export function requireTarget(
  name: string,
  raw: string | undefined,
  env: { ALLOW_PROTECTED_HOST?: string },
): string {
  const d = checkTarget(name, raw, env)
  if (!d.ok) throw new Error(d.reason)
  if (d.warning) console.error(d.warning)
  return d.url as string
}
