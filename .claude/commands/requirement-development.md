Write a new FR-BBxxx requirement document, or revise an existing one if validation findings are provided as argument.

Do NOT write any code.

---

## Step 1 — Read Context

1. Read `corporate_exam_platform_roadmap.md` section for the relevant phase.
2. Read `docs/architecture-guide.md` for existing DB schema and API conventions.
3. Read `docs/requirements/README.md` to find the next available FR-BBxxx number and avoid duplicates.
4. If revising: read the existing requirement doc and any validation findings provided.

## Step 2 — Determine Scope

- Identify all tables, API endpoints, frontend pages, and i18n keys involved.
- Note all dependencies on other FR-BBxxx items.

## Step 3 — Write the Requirement Document

File path: `docs/requirements/FR-BB{xx}-{kebab-case-title}.md`

Use this exact template:

```markdown
# FR-BBxx — {Title}

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BBxx |
| Phase | {N} — {Phase Name} |
| Priority | {1-3} |
| Status | Draft |
| Depends On | {FR-BBxx, or none} |

## Description
{One paragraph describing the feature and its business purpose.}

## Acceptance Criteria
- [ ] AC-1: {testable criterion}
- [ ] AC-2: ...
(Minimum 5 for full-stack features; minimum 3 for backend-only or frontend-only)

## Technical Specification

### Database Schema
{DDL for new tables or columns. Include indexes, foreign keys, constraints.}

### API Contract
{For each endpoint: METHOD /path, auth requirement, request body, success response, error codes.
All responses: `{ data, error: null }` on success; `{ data: null, error: { code, message } }` on failure.}

### Go Implementation Notes
{Package name, handler → service → repository split. Middleware required.}

### Frontend Implementation Notes
{Route path, React Query keys, component structure, shadcn/ui components, i18n key namespaces.}

## Notes
{Out of scope items, known constraints, deferred decisions.}
```

## Step 4 — Update the Requirements Index

Append a row to `docs/requirements/README.md`:
`| FR-BBxx | {title} | {short description} | Draft |`

## Step 5 — Revision Mode

If called with validation findings:
1. Read the existing requirement doc in full.
2. Address every finding — do not skip any severity level.
3. Overwrite the existing doc with the revised version.
4. Do NOT change the FR number.
