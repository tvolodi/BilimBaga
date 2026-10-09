/**
 * FR-BB65 AC-2 - Performance: k6 load test over six endpoint groups.
 *
 * Profile (defaults, overridable): 100 VUs for 2 minutes. Every iteration calls each group once
 * (tagged `ep:<group>`), then sleeps 0.5 s.
 *
 * Thresholds (all must hold, k6 exits non-zero otherwise):
 *   - p(95) http_req_duration < 200 ms for EACH group below
 *   - http_req_failed rate < 1 %
 *   - checks rate >= 99 %
 *
 * Groups (tag ep):
 *   exams         GET /api/v1/exams             admin token (or TEST_TOKEN fallback)
 *   users_me      GET /api/v1/users/me          admin token
 *   tenant_config GET /api/v1/tenant/config     admin token
 *   categories    GET /api/v1/categories        admin token
 *   portal_exams  GET /api/v1/portal/exams      employee token
 *   admin_dash    GET /api/v1/admin/dashboard   admin token
 *
 * Env (no host default on purpose):
 *   BASE_URL (or K6_BASE_URL)  REQUIRED, e.g. http://localhost:8080
 *   K6_ADMIN_EMAIL / K6_ADMIN_PASS        admin credentials (seeded user)
 *   K6_EMPLOYEE_EMAIL / K6_EMPLOYEE_PASS  employee credentials (seeded user)
 *   TEST_TOKEN + ALLOW_PARTIAL=1          fallback JWT for the `exams` group only (other groups skipped,
 *                                         so this is NOT a full AC-2 run)
 *   VUS (100), DURATION (2m)              profile overrides
 *   ALLOW_REMOTE=1                        REQUIRED to target any non-local host (never default it)
 *
 * Safety: refuses any host that is not localhost / *.localhost / 127.0.0.0/8 / ::1 unless
 * ALLOW_REMOTE=1, and ALWAYS refuses bilimbaga-test.ai-dala.com (customer demo, DEC-001).
 * Run it against a LOCAL stack started with DISABLE_RATE_LIMIT=true (test-only API env; without it
 * the per-IP limiters return 429 for 100 VUs from one host). Never set that env on a shared stack.
 *
 * Usage:
 *   k6 run -e BASE_URL=http://localhost:8080 \
 *     -e K6_ADMIN_EMAIL=... -e K6_ADMIN_PASS=... -e K6_EMPLOYEE_EMAIL=... -e K6_EMPLOYEE_PASS=... \
 *     --summary-export=docs/test-reports/k6-summary-<date>.json docs/test-reports/k6-load-test.js
 */
import http from 'k6/http';
import { check, sleep } from 'k6';

const PROTECTED_HOST = 'bilimbaga-test.ai-dala.com';

// Parses the authority of an http(s) URL without relying on the URL global (not in every k6 version).
function parseTarget(raw) {
  const m = /^(https?):\/\/([^/?#]*)([/?#].*)?$/i.exec(raw || '');
  if (!m) throw new Error('BASE_URL must be an absolute http(s) URL, got: ' + JSON.stringify(raw));
  let auth = m[2];
  if (auth.indexOf(String.fromCharCode(92)) !== -1 || /[^\x21-\x7e]/.test(auth) || auth.indexOf('%') !== -1) {
    throw new Error('BASE_URL host contains forbidden characters (refusing to guess): ' + JSON.stringify(raw));
  }
  auth = auth.slice(auth.lastIndexOf('@') + 1); // drop userinfo; the real host is after the last '@'
  let host;
  if (auth.charAt(0) === '[') {
    const end = auth.indexOf(']');
    if (end < 0) throw new Error('BASE_URL has an unterminated IPv6 literal');
    host = auth.slice(1, end);
  } else {
    host = auth.replace(/:\d*$/, '');
  }
  host = host.toLowerCase().replace(/\.+$/, '');
  if (!host) throw new Error('BASE_URL has an empty host');
  return { host: host, base: raw.replace(/\/+$/, '') };
}

function isLocalHost(h) {
  return (
    h === 'localhost' ||
    /\.localhost$/.test(h) ||
    h === '::1' ||
    /^127\.\d{1,3}\.\d{1,3}\.\d{1,3}$/.test(h)
  );
}

function resolveTarget() {
  const raw = __ENV.BASE_URL || __ENV.K6_BASE_URL;
  if (!raw) {
    throw new Error('BASE_URL is required (no default). Example: -e BASE_URL=http://localhost:8080');
  }
  const t = parseTarget(raw);
  if (t.host === PROTECTED_HOST || /\.bilimbaga-test\.ai-dala\.com$/.test(t.host)) {
    throw new Error('Refusing to load-test ' + PROTECTED_HOST + ' (customer demo, DEC-001). No override exists.');
  }
  if (!isLocalHost(t.host) && __ENV.ALLOW_REMOTE !== '1') {
    throw new Error('Refusing non-local host "' + t.host + '". Load tests run against a LOCAL stack; set ALLOW_REMOTE=1 only with explicit owner approval.');
  }
  return t.base;
}

const BASE = resolveTarget();

const GROUPS = {
  exams: { path: '/api/v1/exams', role: 'admin' },
  users_me: { path: '/api/v1/users/me', role: 'admin' },
  tenant_config: { path: '/api/v1/tenant/config', role: 'admin' },
  categories: { path: '/api/v1/categories', role: 'admin' },
  portal_exams: { path: '/api/v1/portal/exams', role: 'employee' },
  admin_dash: { path: '/api/v1/admin/dashboard', role: 'admin' },
};

const thresholds = {
  http_req_failed: ['rate<0.01'],
  checks: ['rate>=0.99'],
};
Object.keys(GROUPS).forEach(function (g) {
  thresholds['http_req_duration{ep:' + g + '}'] = ['p(95)<200'];
});

export const options = {
  vus: parseInt(__ENV.VUS || '100', 10),
  duration: __ENV.DURATION || '2m',
  thresholds: thresholds,
};

function login(email, pass, label) {
  const res = http.post(BASE + '/api/v1/auth/login', JSON.stringify({ email: email, password: pass }), {
    headers: { 'Content-Type': 'application/json' },
    tags: { ep: 'setup_login' },
  });
  let token = null;
  try {
    token = JSON.parse(res.body).data.access_token;
  } catch (e) {
    token = null;
  }
  if (res.status !== 200 || !token) {
    // Never print credentials or the response body.
    throw new Error('setup: ' + label + ' login failed with HTTP ' + res.status);
  }
  return token;
}

export function setup() {
  const adminEmail = __ENV.K6_ADMIN_EMAIL;
  const adminPass = __ENV.K6_ADMIN_PASS;
  const empEmail = __ENV.K6_EMPLOYEE_EMAIL;
  const empPass = __ENV.K6_EMPLOYEE_PASS;
  const tokens = { admin: null, employee: null };

  if (adminEmail && adminPass) {
    tokens.admin = login(adminEmail, adminPass, 'admin');
  } else if (__ENV.TEST_TOKEN) {
    if (__ENV.ALLOW_PARTIAL !== '1') {
      throw new Error('TEST_TOKEN alone exercises only the exams group (the other five would pass vacuously). Set ALLOW_PARTIAL=1 to accept that.');
    }
    tokens.admin = __ENV.TEST_TOKEN; // fallback, only usable for the exams group
    tokens.adminFallbackOnly = true;
  } else {
    throw new Error('Provide K6_ADMIN_EMAIL + K6_ADMIN_PASS (or TEST_TOKEN for the exams group only).');
  }
  if (empEmail && empPass) {
    tokens.employee = login(empEmail, empPass, 'employee');
  } else if (!tokens.adminFallbackOnly) {
    // A silently skipped group would pass its threshold vacuously, so a full run needs both users.
    throw new Error('Provide K6_EMPLOYEE_EMAIL + K6_EMPLOYEE_PASS (portal_exams group needs an employee token).');
  }
  return tokens;
}

function hit(name, token) {
  const res = http.get(BASE + GROUPS[name].path, {
    headers: { Authorization: 'Bearer ' + token },
    tags: { ep: name },
  });
  check(res, {
    [name + ' status 200']: (r) => r.status === 200,
    [name + ' envelope ok']: (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.data !== undefined && body.error === null;
      } catch (e) {
        return false;
      }
    },
  });
}

export default function (tokens) {
  Object.keys(GROUPS).forEach(function (name) {
    const token = GROUPS[name].role === 'employee' ? tokens.employee : tokens.admin;
    if (!token) return; // only reachable in TEST_TOKEN fallback mode
    if (tokens.adminFallbackOnly && name !== 'exams') return; // TEST_TOKEN only covers /exams
    hit(name, token);
  });
  sleep(0.5);
}
