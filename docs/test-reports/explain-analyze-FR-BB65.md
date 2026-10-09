# FR-BB65 EXPLAIN ANALYZE evidence (E1, AC-1) - 2026-10-09

Issue: #32 (UAT evidence plan), requirement `docs/requirements/FR-BB65.Performance.md`, migration `024_performance_indexes`.
Run by: bb-uat. Scope: **local, isolated throwaway database only** (compose project `bbf-perf`, own volume, no QA/test host touched).

## Environment
- PostgreSQL 16.11 (alpine, docker), default settings: `shared_buffers=128MB`, `work_mem=4MB`, `random_page_cost=4`.
- Schema from the repository migrations (001..033 applied by the API at startup on a fresh volume), then a synthetic perf seed (SQL below), then `ANALYZE`.
- Row counts: users 5,001; questions 10,000; question_translations 30,000; exam_assignments 200,000; exam_sessions 60,000; session_answers 600,000; certificates 18,000; audit_log 400,000.
- Each query was run once with `EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)` after the seed (warm cache after the seed, single connection, no concurrent load). Parameter values were picked from existing rows with `\gset`.

## Result (pass threshold E1: each query shows its named index)
| # | Query pattern | Index in plan | Plan node | Execution time |
|---|---------------|---------------|-----------|----------------|
| Q1 | `session_answers WHERE session_id = $1` | `idx_session_answers_session_id` | Bitmap Index Scan -> Bitmap Heap Scan (10 rows) | 0.23 ms |
| Q2a | `audit_log ORDER BY created_at DESC LIMIT 50` | `idx_audit_log_created_at_actor` | Index Scan (50 rows, no sort) | 0.14 ms |
| Q2b | `audit_log WHERE actor_id = $1 AND created_at >= now()-7d ORDER BY created_at DESC LIMIT 50` | `idx_audit_log_actor_id` (NOT the composite) | Bitmap Index Scan (80 rows) + sort | 0.77 ms |
| Q3 | `exam_sessions WHERE user_id=$1 AND exam_id=$2 AND status='submitted'` | `idx_exam_sessions_user_exam_status` | Index Scan (12 rows) | 0.05 ms |
| Q4 | `question_translations WHERE question_id=$1 AND locale='ru'` | `idx_question_translations_question_locale` | Index Scan (1 row) | 0.07 ms |
| Q5 | `exam_assignments WHERE exam_id=$1` | `idx_exam_assignments_exam_id` | Bitmap Index Scan -> Heap Scan (5,000 rows) | 34.1 ms |
| Q6 | `certificates WHERE session_id=$1` | `idx_certificates_session_id` | Index Scan (1 row) | 0.09 ms |
| Q7 | `certificates WHERE verification_code=$1` | `idx_certificates_verification_code` | Index Scan (1 row) | 0.04 ms |

**Verdict: AC-1 evidence met for all seven required indexes** (each of the seven named indexes appears in the plan of the query it was built for; no sequential scan on any of the six tables).

Notes and caveats (honest reading):
- Q2b: for a selective actor the planner chose the single-column `idx_audit_log_actor_id` instead of the composite `(created_at DESC, actor_id)`; the composite index is used by the pure time-ordered query Q2a. Both are fine at this volume; if the composite index is meant to serve the actor+time filter, check the planner choice with production-like statistics.
- Q5 returns 5,000 rows because the synthetic seed assigns every one of 5,000 users to each exam (5,000 assignments per exam); 34 ms is dominated by returning and heap-fetching those rows, not by the index lookup (1.8 ms).
- The data is synthetic (uniform), run on a developer machine, single run each, default Postgres memory settings: it proves index usage and plan shape, not production latency.

## Raw output
```
Pager usage is off.
=== Q1 session_answers by session_id (expect idx_session_answers_session_id)
                                              QUERY PLAN                                              
------------------------------------------------------------------------------------------------------
 Bitmap Heap Scan on session_answers (actual time=0.067..0.169 rows=10 loops=1)
   Recheck Cond: (session_id = 'd59f4777-5634-41b5-a5db-d7f6a62de547'::uuid)
   Heap Blocks: exact=10
   Buffers: shared read=13
   ->  Bitmap Index Scan on idx_session_answers_session_id (actual time=0.040..0.040 rows=10 loops=1)
         Index Cond: (session_id = 'd59f4777-5634-41b5-a5db-d7f6a62de547'::uuid)
         Buffers: shared read=3
 Planning:
   Buffers: shared hit=58 read=6 dirtied=3
 Planning Time: 0.688 ms
 Execution Time: 0.231 ms
(11 rows)

=== Q2a audit_log newest 50 (expect idx_audit_log_created_at_actor)
                                                  QUERY PLAN                                                   
---------------------------------------------------------------------------------------------------------------
 Limit (actual time=0.113..0.119 rows=50 loops=1)
   Buffers: shared hit=1 read=3
   ->  Index Scan using idx_audit_log_created_at_actor on audit_log (actual time=0.112..0.115 rows=50 loops=1)
         Buffers: shared hit=1 read=3
 Planning:
   Buffers: shared hit=26 dirtied=1
 Planning Time: 0.305 ms
 Execution Time: 0.137 ms
(8 rows)

=== Q2b audit_log by actor in the last 7 days, newest first (expect idx_audit_log_created_at_actor or actor_id index)
                                                QUERY PLAN                                                
----------------------------------------------------------------------------------------------------------
 Limit (actual time=0.707..0.714 rows=50 loops=1)
   Buffers: shared hit=44 read=42
   ->  Sort (actual time=0.706..0.709 rows=50 loops=1)
         Sort Key: created_at DESC
         Sort Method: quicksort  Memory: 35kB
         Buffers: shared hit=44 read=42
         ->  Bitmap Heap Scan on audit_log (actual time=0.114..0.666 rows=80 loops=1)
               Recheck Cond: (actor_id = '7a5ca3da-0024-4ad5-a619-1dface465446'::uuid)
               Filter: (created_at >= (now() - '7 days'::interval))
               Heap Blocks: exact=80
               Buffers: shared hit=41 read=42
               ->  Bitmap Index Scan on idx_audit_log_actor_id (actual time=0.060..0.061 rows=80 loops=1)
                     Index Cond: (actor_id = '7a5ca3da-0024-4ad5-a619-1dface465446'::uuid)
                     Buffers: shared read=3
 Planning:
   Buffers: shared hit=10
 Planning Time: 0.183 ms
 Execution Time: 0.774 ms
(18 rows)

=== Q3 exam_sessions by user, exam, status (expect idx_exam_sessions_user_exam_status)
                                                                                    QUERY PLAN                                                                                    
----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
 Index Scan using idx_exam_sessions_user_exam_status on exam_sessions (actual time=0.024..0.037 rows=12 loops=1)
   Index Cond: ((user_id = '1b9b6856-3a4d-49ce-8653-579056475b32'::uuid) AND (exam_id = '948c266c-1b98-41be-a2e0-0d1b999201ec'::uuid) AND (status = 'submitted'::session_status))
   Buffers: shared hit=12 read=2
 Planning:
   Buffers: shared hit=21 read=1 dirtied=2
 Planning Time: 0.270 ms
 Execution Time: 0.052 ms
(7 rows)

=== Q4 question_translations by question and locale (expect idx_question_translations_question_locale)
                                                          QUERY PLAN                                                           
-------------------------------------------------------------------------------------------------------------------------------
 Index Scan using idx_question_translations_question_locale on question_translations (actual time=0.036..0.036 rows=1 loops=1)
   Index Cond: ((question_id = '68c5e872-1cc3-4a9d-a131-f738c337b752'::uuid) AND (locale = 'ru'::text))
   Buffers: shared hit=3
 Planning:
   Buffers: shared hit=15 read=2 dirtied=1
 Planning Time: 0.594 ms
 Execution Time: 0.070 ms
(7 rows)

=== Q5 exam_assignments by exam (expect idx_exam_assignments_exam_id)
                                              QUERY PLAN                                              
------------------------------------------------------------------------------------------------------
 Bitmap Heap Scan on exam_assignments (actual time=2.042..33.726 rows=5000 loops=1)
   Recheck Cond: (exam_id = '948c266c-1b98-41be-a2e0-0d1b999201ec'::uuid)
   Heap Blocks: exact=2667
   Buffers: shared read=2672
   ->  Bitmap Index Scan on idx_exam_assignments_exam_id (actual time=1.791..1.792 rows=5000 loops=1)
         Index Cond: (exam_id = '948c266c-1b98-41be-a2e0-0d1b999201ec'::uuid)
         Buffers: shared read=5
 Planning:
   Buffers: shared hit=92 read=6
 Planning Time: 0.768 ms
 Execution Time: 34.115 ms
(11 rows)

=== Q6 certificates by session (expect idx_certificates_session_id)
                                               QUERY PLAN                                               
--------------------------------------------------------------------------------------------------------
 Index Scan using idx_certificates_session_id on certificates (actual time=0.068..0.069 rows=1 loops=1)
   Index Cond: (session_id = '7ef9b05a-8081-47b0-8a41-f3b9da257266'::uuid)
   Buffers: shared hit=1 read=2
 Planning:
   Buffers: shared hit=15
 Planning Time: 0.150 ms
 Execution Time: 0.086 ms
(7 rows)

=== Q7 certificates by verification_code (expect idx_certificates_verification_code)
                                                  QUERY PLAN                                                   
---------------------------------------------------------------------------------------------------------------
 Index Scan using idx_certificates_verification_code on certificates (actual time=0.030..0.031 rows=1 loops=1)
   Index Cond: (verification_code = '02d72cc2-f4c7-46a6-8251-506db0e38ae4'::uuid)
   Buffers: shared hit=1 read=2
 Planning Time: 0.037 ms
 Execution Time: 0.041 ms
(5 rows)
```

## Queries executed
```sql
\set ON_ERROR_STOP on
\pset pager off
SELECT id AS sid FROM exam_sessions WHERE status='submitted' ORDER BY id OFFSET 30000 LIMIT 1 \gset
SELECT user_id AS uid, exam_id AS eid FROM exam_sessions WHERE id = :'sid' \gset
SELECT actor_id AS aid FROM audit_log WHERE actor_id IS NOT NULL ORDER BY created_at OFFSET 200000 LIMIT 1 \gset
SELECT question_id AS qid FROM question_translations ORDER BY question_id OFFSET 12345 LIMIT 1 \gset
SELECT session_id AS csid, verification_code AS ccode FROM certificates ORDER BY id OFFSET 9000 LIMIT 1 \gset

\echo '=== Q1 session_answers by session_id (expect idx_session_answers_session_id)'
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) SELECT * FROM session_answers WHERE session_id = :'sid';
\echo '=== Q2a audit_log newest 50 (expect idx_audit_log_created_at_actor)'
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) SELECT * FROM audit_log ORDER BY created_at DESC LIMIT 50;
\echo '=== Q2b audit_log by actor in the last 7 days, newest first (expect idx_audit_log_created_at_actor or actor_id index)'
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) SELECT * FROM audit_log WHERE actor_id = :'aid' AND created_at >= now() - interval '7 days' ORDER BY created_at DESC LIMIT 50;
\echo '=== Q3 exam_sessions by user, exam, status (expect idx_exam_sessions_user_exam_status)'
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) SELECT * FROM exam_sessions WHERE user_id = :'uid' AND exam_id = :'eid' AND status = 'submitted';
\echo '=== Q4 question_translations by question and locale (expect idx_question_translations_question_locale)'
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) SELECT * FROM question_translations WHERE question_id = :'qid' AND locale = 'ru';
\echo '=== Q5 exam_assignments by exam (expect idx_exam_assignments_exam_id)'
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) SELECT * FROM exam_assignments WHERE exam_id = :'eid';
\echo '=== Q6 certificates by session (expect idx_certificates_session_id)'
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) SELECT * FROM certificates WHERE session_id = :'csid';
\echo '=== Q7 certificates by verification_code (expect idx_certificates_verification_code)'
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) SELECT * FROM certificates WHERE verification_code = :'ccode';
```

## Perf seed (isolated DB only; never run on a shared database)
```sql
-- FR-BB65 perf seed for an ISOLATED throwaway DB (never run on a shared/QA/test database).
-- Volumes: 5k users, 40 exams, 10k questions (+30k translations), 200k assignments,
-- 60k sessions, 600k session answers, 20k certificates, 400k audit rows.
\set ON_ERROR_STOP on
\timing off
BEGIN;
INSERT INTO users (email, password_hash, full_name, role_id, status)
SELECT 'perf'||g||'@perf.test', 'x', 'Perf User '||g, (SELECT id FROM roles WHERE name='employee'), 'active'
FROM generate_series(1,5000) g;

INSERT INTO exams (title, time_limit_minutes, passing_score_pct, created_by, status)
SELECT 'Perf Exam '||g, 30, 70, (SELECT id FROM users WHERE email='admin@bilimbaga.local'), 'active'
FROM generate_series(1,40) g;

INSERT INTO questions (category_id, difficulty, type, created_by, status, default_locale)
SELECT (SELECT id FROM categories ORDER BY id LIMIT 1), (ARRAY['easy','medium','hard'])[1+g%3], 'single',
       (SELECT id FROM users WHERE email='admin@bilimbaga.local'), 'active', 'en'
FROM generate_series(1,10000) g;

INSERT INTO question_translations (question_id, locale, stem)
SELECT q.id, l.loc, 'Perf stem '||l.loc||' '||q.id
FROM questions q CROSS JOIN (VALUES ('en'),('ru'),('kk')) AS l(loc);

INSERT INTO exam_assignments (exam_id, assignee_type, assignee_id, assigned_by)
SELECT e.id, 'user', u.id, (SELECT id FROM users WHERE email='admin@bilimbaga.local')
FROM (SELECT id, row_number() OVER () rn FROM exams WHERE title LIKE 'Perf Exam%') e
JOIN (SELECT id, row_number() OVER () rn FROM users WHERE email LIKE 'perf%') u ON u.rn <= 5000;

INSERT INTO exam_sessions (exam_id, user_id, status, seed, expires_at, started_at, submitted_at, score_pct, passed)
SELECT e.id, u.id, (ARRAY['submitted','submitted','submitted','auto_submitted','grading_pending'])[1+g%5]::session_status, g,
       now() + interval '1 day', now() - (g%90||' days')::interval, now() - (g%90||' days')::interval + interval '20 minutes',
       (g%100)::numeric, (g%100)>=70
FROM generate_series(1,60000) g
JOIN LATERAL (SELECT id FROM exams WHERE title LIKE 'Perf Exam%' OFFSET (g%40) LIMIT 1) e ON true
JOIN LATERAL (SELECT id FROM users WHERE email LIKE 'perf%' OFFSET (g%5000) LIMIT 1) u ON true;

INSERT INTO session_answers (session_id, question_id, selected_option_ids, saved_at, time_spent_seconds)
SELECT s.id, q.id, '[]'::jsonb, now(), 5
FROM (SELECT id, row_number() OVER () rn FROM exam_sessions) s
CROSS JOIN generate_series(0,9) k
JOIN (SELECT id, row_number() OVER () rn FROM questions) q ON q.rn = 1 + ((s.rn*7 + k*13) % 10000);

INSERT INTO certificates (session_id, employee_name, exam_title, score_pct, template_snapshot)
SELECT id, 'Perf Employee', 'Perf Exam', score_pct, '{}'::jsonb
FROM exam_sessions WHERE passed ORDER BY id LIMIT 20000;

INSERT INTO audit_log (tenant_id, actor_id, action, entity_type, ip, metadata, created_at)
SELECT 'public', (SELECT id FROM users WHERE email LIKE 'perf%' OFFSET (g%5000) LIMIT 1), (ARRAY['auth.login','users.create','exam.publish','session.submit'])[1+g%4],
       'user', '10.0.0.1', '{}'::jsonb, now() - (g||' seconds')::interval
FROM generate_series(1,400000) g;
COMMIT;
ANALYZE;
SELECT 'users', count(*) FROM users UNION ALL SELECT 'questions', count(*) FROM questions UNION ALL SELECT 'question_translations', count(*) FROM question_translations
UNION ALL SELECT 'exam_assignments', count(*) FROM exam_assignments UNION ALL SELECT 'exam_sessions', count(*) FROM exam_sessions UNION ALL SELECT 'session_answers', count(*) FROM session_answers
UNION ALL SELECT 'certificates', count(*) FROM certificates UNION ALL SELECT 'audit_log', count(*) FROM audit_log;
```
