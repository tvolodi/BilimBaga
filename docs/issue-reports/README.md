# Issue Reports — Knowledge Base Index

This index is maintained automatically by the Issue Resolution agent after every resolved issue.
Agents: search this table first to check for prior art before opening a new ISS file.

## How to Search

1. Scan the **Module** and **Tags** columns for the affected area.
2. If a row matches, read the linked file — if `Status = resolved` and the root cause matches your bug, treat it as a recurrence.
3. If no row matches, create a new `ISS-{NNN}-{slug}.md` file using the template in [issue-resolution.agent.md](../../.github/agents/issue-resolution.agent.md).

## Index

| ID | Title | Module | Severity | Status | Recurrences | Resolved |
|----|-------|--------|----------|--------|-------------|----------|
| [ISS-001](ISS-001-stale-dist-usematches-crash.md) | Stale Docker dist serves old Breadcrumb with useMatches — crashes on BrowserRouter | auth | high | resolved | 1 | 2026-05-17 |
| [ISS-002](ISS-002-e2e-walkthrough-false-positive-passes.md) | E2E full walkthrough test has systematic false-positive passes | frontend | high | resolved | 1 | 2026-05-17 |

<!-- Agents: append rows here in the format above after each resolution. -->
<!-- Example: | [ISS-001](ISS-001-jwt-token-not-refreshed.md) | JWT token not refreshed on expiry | auth | high | resolved | 1 | 2026-06-01 | -->

## Recurring Issues (≥ 2 occurrences)

<!-- Agents: list issues where recurrence_count >= 2. These signal incomplete fixes or missing regression tests. -->

| ID | Title | Recurrences | Root Cause Summary | Regression Test |
|----|-------|-------------|-------------------|-----------------|

## Pattern Summary

<!-- Agents: update this section when any module accumulates ≥ 3 issues. -->

| Module | Issue Count | Most Common Root Cause |
|--------|-------------|------------------------|
