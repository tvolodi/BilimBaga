# FR-BB65 AC-1 EXPLAIN ANALYZE evidence (local stack, deb2165)

Data volume is tiny on the local stack, so timings are sub-millisecond and show plan shape only. A sequential scan on a subquery picking a sample key is expected; the named index is used by the outer query in every case.

```
== Q1 session_answers by session_id (idx_session_answers_session_id)
                                                                 QUERY PLAN                                                                  
---------------------------------------------------------------------------------------------------------------------------------------------
 Bitmap Heap Scan on session_answers  (cost=4.20..11.31 rows=3 width=124) (actual time=0.066..0.068 rows=1 loops=1)
   Recheck Cond: (session_id = $0)
   Heap Blocks: exact=1
   Buffers: shared hit=3
   InitPlan 1 (returns $0)
     ->  Limit  (cost=0.00..0.03 rows=1 width=16) (actual time=0.040..0.040 rows=1 loops=1)
           Buffers: shared hit=1
           ->  Seq Scan on session_answers session_answers_1  (cost=0.00..15.30 rows=530 width=16) (actual time=0.019..0.019 rows=1 loops=1)
                 Buffers: shared hit=1
   ->  Bitmap Index Scan on idx_session_answers_session_id  (cost=0.00..4.17 rows=3 width=0) (actual time=0.058..0.058 rows=1 loops=1)
         Index Cond: (session_id = $0)
         Buffers: shared hit=2
 Planning:
   Buffers: shared hit=101
 Planning Time: 1.159 ms
 Execution Time: 0.158 ms
(16 rows)

== Q2 audit_log latest by created_at, actor (idx_audit_log_created_at_actor)
                                                                      QUERY PLAN                                                                      
------------------------------------------------------------------------------------------------------------------------------------------------------
 Limit  (cost=0.27..7.04 rows=50 width=146) (actual time=0.038..0.058 rows=50 loops=1)
   Buffers: shared hit=7 dirtied=3
   ->  Index Scan using idx_audit_log_created_at_actor on audit_log  (cost=0.27..60.37 rows=444 width=146) (actual time=0.037..0.054 rows=50 loops=1)
         Buffers: shared hit=7 dirtied=3
 Planning:
   Buffers: shared hit=185 dirtied=6
 Planning Time: 1.329 ms
 Execution Time: 0.079 ms
(8 rows)

== Q3 exam_sessions by user, exam, status (idx_exam_sessions_user_exam_status)
                                                                     QUERY PLAN                                                                     
----------------------------------------------------------------------------------------------------------------------------------------------------
 Index Scan using idx_exam_sessions_user_exam_status on exam_sessions  (cost=0.21..8.23 rows=1 width=137) (actual time=0.054..0.056 rows=1 loops=1)
   Index Cond: ((user_id = $0) AND (exam_id = $1) AND (status = 'submitted'::session_status))
   Buffers: shared hit=4
   InitPlan 1 (returns $0)
     ->  Limit  (cost=0.00..0.03 rows=1 width=16) (actual time=0.019..0.019 rows=1 loops=1)
           Buffers: shared hit=1
           ->  Seq Scan on exam_sessions exam_sessions_1  (cost=0.00..14.90 rows=490 width=16) (actual time=0.018..0.018 rows=1 loops=1)
                 Buffers: shared hit=1
   InitPlan 2 (returns $1)
     ->  Limit  (cost=0.00..0.03 rows=1 width=16) (actual time=0.001..0.001 rows=1 loops=1)
           Buffers: shared hit=1
           ->  Seq Scan on exam_sessions exam_sessions_2  (cost=0.00..14.90 rows=490 width=16) (actual time=0.001..0.001 rows=1 loops=1)
                 Buffers: shared hit=1
 Planning:
   Buffers: shared hit=178
 Planning Time: 1.960 ms
 Execution Time: 0.109 ms
(17 rows)

== Q4 question_translations by question, locale (idx_question_translations_question_locale)
                                                                            QUERY PLAN                                                                             
-------------------------------------------------------------------------------------------------------------------------------------------------------------------
 Index Scan using idx_question_translations_question_locale on question_translations  (cost=0.18..8.20 rows=1 width=120) (actual time=0.056..0.057 rows=1 loops=1)
   Index Cond: ((question_id = $0) AND (locale = 'en'::text))
   Buffers: shared hit=3
   InitPlan 1 (returns $0)
     ->  Limit  (cost=0.00..0.03 rows=1 width=16) (actual time=0.027..0.028 rows=1 loops=1)
           Buffers: shared hit=1
           ->  Seq Scan on question_translations question_translations_1  (cost=0.00..15.50 rows=550 width=16) (actual time=0.027..0.027 rows=1 loops=1)
                 Buffers: shared hit=1
 Planning:
   Buffers: shared hit=88
 Planning Time: 0.768 ms
 Execution Time: 0.075 ms
(12 rows)

== Q5 exam_assignments by exam (idx_exam_assignments_exam_id)
                                                                  QUERY PLAN                                                                   
-----------------------------------------------------------------------------------------------------------------------------------------------
 Bitmap Heap Scan on exam_assignments  (cost=4.20..12.67 rows=4 width=84) (actual time=0.027..0.028 rows=3 loops=1)
   Recheck Cond: (exam_id = $0)
   Heap Blocks: exact=1
   Buffers: shared hit=3 dirtied=1
   InitPlan 1 (returns $0)
     ->  Limit  (cost=0.00..0.02 rows=1 width=16) (actual time=0.013..0.013 rows=1 loops=1)
           Buffers: shared hit=1 dirtied=1
           ->  Seq Scan on exam_assignments exam_assignments_1  (cost=0.00..17.20 rows=720 width=16) (actual time=0.012..0.013 rows=1 loops=1)
                 Buffers: shared hit=1 dirtied=1
   ->  Bitmap Index Scan on idx_exam_assignments_exam_id  (cost=0.00..4.18 rows=4 width=0) (actual time=0.021..0.021 rows=6 loops=1)
         Index Cond: (exam_id = $0)
         Buffers: shared hit=2 dirtied=1
 Planning:
   Buffers: shared hit=94
 Planning Time: 0.615 ms
 Execution Time: 0.057 ms
(16 rows)

== Q6 certificates by session (idx_certificates_session_id)
                                                                 QUERY PLAN                                                                 
--------------------------------------------------------------------------------------------------------------------------------------------
 Index Scan using idx_certificates_session_id on certificates  (cost=0.18..8.20 rows=1 width=164) (actual time=0.025..0.026 rows=1 loops=1)
   Index Cond: (session_id = $0)
   Buffers: shared hit=3
   InitPlan 1 (returns $0)
     ->  Limit  (cost=0.00..0.03 rows=1 width=16) (actual time=0.007..0.008 rows=1 loops=1)
           Buffers: shared hit=1
           ->  Seq Scan on certificates certificates_1  (cost=0.00..14.20 rows=420 width=16) (actual time=0.006..0.007 rows=1 loops=1)
                 Buffers: shared hit=1
 Planning:
   Buffers: shared hit=80
 Planning Time: 0.716 ms
 Execution Time: 0.051 ms
(12 rows)

== Q7 certificates by verification code (idx_certificates_verification_code)
                                                                    QUERY PLAN                                                                     
---------------------------------------------------------------------------------------------------------------------------------------------------
 Index Scan using idx_certificates_verification_code on certificates  (cost=0.18..8.20 rows=1 width=164) (actual time=0.071..0.073 rows=1 loops=1)
   Index Cond: (verification_code = $0)
   Buffers: shared hit=3
   InitPlan 1 (returns $0)
     ->  Limit  (cost=0.00..0.03 rows=1 width=16) (actual time=0.006..0.007 rows=1 loops=1)
           Buffers: shared hit=1
           ->  Seq Scan on certificates certificates_1  (cost=0.00..14.20 rows=420 width=16) (actual time=0.006..0.006 rows=1 loops=1)
                 Buffers: shared hit=1
 Planning Time: 0.067 ms
 Execution Time: 0.090 ms
(10 rows)

```
