---
slug: anti-cheat-hygiene-fr-bb319
title: "Anti-Cheat Event Hygiene (single report, submit only on tab switch, fullscreen exit) — UAT Scenario"
feature: FR-BB319 (amends FR-BB38 AC-10 and FR-BB314 AC-7); issue #421; implementation #400
version: 1
created: 2026-10-10
author: UAT (Claude Code)
---

Target: local (default). Never the production-class demo instance (DEC-001).

## Code that must be on main before running

- FR-BB319 implementation merged (issue #400, PR merged as eb562132 or later). Server `ReportEvent` caps `blur` and `fullscreen_exit` at `warn`; client hook ordered rules; fullscreen requested in `EmployeePortal` `handleConfirm`.
- Frontend rebuilt (`npm run build` in the served image) and API rebuilt. No migration.

## Preconditions

| Account | Role | Password | Source |
|---------|------|----------|--------|
| `admin@bilimbaga.local` | super_admin | current admin password | seeded |
| `employee@bilimbaga.local` | employee | `Employee1234!` | E2E fixtures |

Exams (created by UAT in S0, published and assigned to the employee):
- `EXAM-LOG` with `on_tab_switch = 'log'`
- `EXAM-WARN` with `on_tab_switch = 'warn'`
- `EXAM-SUBMIT` with `on_tab_switch = 'submit'`

Platform at `http://localhost` (nginx). API at `http://localhost/api/v1`.

## S0: Setup

| Step | Actor | Action | Expected | Pass/Fail |
|------|-------|--------|----------|-----------|
| 1 | Admin | Create three published exams with the policies above and assign each to the employee | 201 each; assignments exist | |
| 2 | Tester | Record the main SHA and the frontend/API build used | Recorded in the report | |

## S1: API policy matrix (per event type)

Start one session per exam for the employee (`POST /portal/exams/{id}/sessions`), then send events with `POST /portal/sessions/{sid}/events {type}`.

| Step | Exam policy | Event | Expected response | Expected stored `action_taken` | Session status after |
|------|-------------|-------|-------------------|--------------------------------|----------------------|
| 1 | submit | `blur` | 200 `{warn:true, event_count:1}`; no `session_id`, `status`, `score_pct` | `warn` | `in_progress` |
| 2 | submit | `fullscreen_exit` | 200 `{warn:true, event_count:2}` | `warn` | `in_progress` |
| 3 | submit | `tab_switch` | 200 with `status: auto_submitted` | `submit` | `auto_submitted` |
| 4 | warn | `blur` | 200 `{warn:true}` | `warn` | `in_progress` |
| 5 | log | `blur` | 200 `{warn:false}` | `log` | `in_progress` |

Check the stored rows with `SELECT type-less action_taken FROM tab_switch_events WHERE session_id = ...` (the type is not stored; the limitation is noted in FR-BB319).

## S2: Headed browser, fullscreen exit path (exam start click)

Run headed (not headless), fresh browser context.

| Step | Actor | Action | Expected | Pass/Fail |
|------|-------|--------|----------|-----------|
| 1 | Employee | Log in, open `/portal`, click start on `EXAM-WARN`, confirm in the start dialog | Session page `/portal/sessions/{id}` opens; `document.fullscreenElement` is not null (fullscreen requested inside the click, AC-8) | |
| 2 | Tester | Exit fullscreen (`document.exitFullscreen()` or Esc) | Exactly one `POST .../events` with `type: 'fullscreen_exit'` (not a `blur` for the same loss, AC-4/5/7) | |
| 3 | Employee | Observe the page | `TabSwitchWarningModal` shows the running count `Violations recorded: 1` and a `Return to full screen` button (AC-9/10) | |
| 4 | Employee | Click `Return to full screen` | `document.fullscreenElement` is not null again; modal behaviour as designed | |
| 5 | Employee | Repeat steps 1-3 on `EXAM-SUBMIT` | The session stays `in_progress` after the fullscreen exit (no auto-submit); warn modal shown | |
| 6 | Tester | Screenshot each step; keep the video | Attached to the report | |

## S3: Single report per incident (client rules)

Headed, on `EXAM-WARN`:
- Switch to another window (or simulate `visibilitychange` hidden): exactly one `tab_switch`, no duplicate `blur`.
- Click outside the window only (window `blur` without hidden): exactly one `blur`.

Count the requests with `waitForRequest` on the events URL and `postDataJSON().type`.

## S4: Log policy does not interrupt

On `EXAM-LOG`: fullscreen exit sends one `fullscreen_exit`; no warn modal; the exam continues.

## Pass / fail criteria

- PASS: S1 rows match the table; S2 steps 1-5 as expected in a headed browser; S3 counts are exactly one per incident; S4 no modal.
- FAIL (defect): `blur` or `fullscreen_exit` submits under `on_tab_switch = submit`; `action_taken` stores the raw policy instead of the effective one; a single focus loss produces two events; fullscreen not requested from the start click; no modal on warn.
- ENV ISSUE: the browser cannot enter fullscreen (headless or no display); report as blocked, not failed.

## Acceptance coverage

| FR-BB319 AC | Covered by |
|-------------|-----------|
| AC-1 submit does not submit on blur/fullscreen_exit | S1 steps 1-2 |
| AC-2 warn/log unchanged | S1 steps 4-5, S4 |
| AC-3 action_taken = effective policy | S1 stored column |
| AC-4 to AC-7 ordered rules, single report | S3 |
| AC-8 fullscreen requested at start click | S2 step 1 |
| AC-9 to AC-10 modal count and return button | S2 steps 3-4 |
