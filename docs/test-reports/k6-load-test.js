/**
 * FR-BB65 — Performance: k6 load test
 *
 * Simulates 100 concurrent virtual users for 2 minutes against the exam list
 * endpoint. Acceptance threshold: p95 response time < 200ms.
 *
 * Usage:
 *   k6 run --env TEST_TOKEN=<jwt> docs/test-reports/k6-load-test.js
 */
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  vus: 100,
  duration: '2m',
  thresholds: {
    'http_req_duration': ['p(95)<200'],
    'http_req_failed': ['rate<0.01'], // less than 1% error rate
  },
};

export default function () {
  const res = http.get('http://localhost:8080/api/v1/exams', {
    headers: {
      Authorization: `Bearer ${__ENV.TEST_TOKEN}`,
    },
  });

  check(res, {
    'status 200': (r) => r.status === 200,
    'response has data': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.data !== undefined && body.error === null;
      } catch {
        return false;
      }
    },
  });

  sleep(0.5);
}
