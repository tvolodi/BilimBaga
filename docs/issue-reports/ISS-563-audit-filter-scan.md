# ISS-563: audit filter options scan the tenant's log on every call

Issue: #563 (from the architect review of PR 562 / #557).
Endpoint: `GET /audit/actions`, served by `audit.ListFilterOptions` (`backend/internal/audit/repository.go`). It runs two
queries per call, one per column:

```sql
SELECT DISTINCT <col> COLLATE "C" AS value FROM audit_log
WHERE tenant_id = $1 AND <col> IS NOT NULL AND <col> <> '' ORDER BY value
```

with `<col>` = `action`, then `entity_type`.

## Measurement

Throwaway `postgres:16` (cached image, `--pull never`, its own container and port), with the production migrations applied
in a test schema. The seed is 1,010,000 rows: `tenant-a` 800,000 (80%), `tenant-b` 200,000, `tenant-small` 10,000 (1%).
`ANALYZE` ran before each measurement. The uat stack was not touched.

| Tenant | Indexes | Plan per scan (EXPLAIN ANALYZE) | Per scan | `ListFilterOptions` per call |
|---|---|---|---|---|
| tenant-a (80%) | current | Parallel Seq Scan on audit_log, Filter on `tenant_id` | 57-68 ms | 111-128 ms |
| tenant-a (80%) | + (tenant_id, action), (tenant_id, entity_type) | Parallel Seq Scan (same plan) | 61-66 ms | 112-124 ms |
| tenant-small (1%) | current | Bitmap Index Scan on `idx_audit_log_tenant_created` | 1.9 ms | 4-5 ms |
| tenant-small (1%) | + composite | Bitmap Index Scan on the new index | 1.8-1.9 ms | 4.6 ms |

## Decision

No index migration. The composite indexes do not change the plan for the large tenant: the tenant holds most of the rows,
so a sequential scan is cheaper. For the small tenant the current index already gives the same result. Each audit insert
would pay for two more b-tree indexes on a write-heavy table.

Instead the audit service caches the filter options per tenant for 5 minutes (`filterOptionsTTL`). Filter options can be a
few minutes stale without harm, and a cache hit costs no query. Errors are not cached. The cache is per API instance, so
each instance reads the log at most once per TTL and tenant.

No migration, so the migration lock is not taken and this is not T4.

## Limits

- A new action code or entity type can appear in the filter list up to 5 minutes after its first audit row.
- The cache holds one entry per tenant. Tenants are few, so there is no eviction.
- The first call after a restart, or after the TTL, still runs the two scans (about 115 ms at 1M rows).
