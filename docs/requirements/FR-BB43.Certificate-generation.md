# FR-BB43 — Certificate Generation

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB43 |
| Phase | 4 — Results & Certificates |
| Priority | 1 |
| Status | Draft |
| Depends On | FR-BB41 |

## Description
Manages the lifecycle of completion certificates. When an employee passes an exam with `certificate_enabled = true`, they can request a PDF certificate via an authenticated endpoint. The first request lazily creates a `certificates` record (capturing a snapshot of tenant branding and result data) and streams a generated PDF. A public verification endpoint allows anyone to confirm certificate authenticity using the unique verification code embedded in the QR code.

## Acceptance Criteria
- [ ] AC-1: `GET /portal/sessions/:id/certificate` returns HTTP 403 if the session does not belong to the authenticated user.
- [ ] AC-2: `GET /portal/sessions/:id/certificate` returns HTTP 422 with `code: EXAM_NOT_CERTIFIABLE` if `exam.certificate_enabled = false`.
- [ ] AC-3: `GET /portal/sessions/:id/certificate` returns HTTP 422 with `code: SESSION_NOT_PASSED` if `exam_sessions.passed = false` or `passed` is null.
- [ ] AC-4: `GET /portal/sessions/:id/certificate` returns HTTP 422 with `code: SESSION_NOT_SUBMITTED` if session status is not `submitted`.
- [ ] AC-5: On the first certificate request, a row is inserted into `certificates` with a `verification_code` (UUID v4), capturing `employee_name`, `exam_title`, `score_pct`, and `template_snapshot` (company branding) at that instant; subsequent requests reuse the existing record but regenerate the PDF bytes on the fly.
- [ ] AC-6: The response sets `Content-Type: application/pdf` and `Content-Disposition: attachment; filename="certificate-{verification_code}.pdf"`.
- [ ] AC-7: `GET /verify/:code` is a public endpoint requiring no authentication; it returns HTTP 200 with `{ valid: true, ... }` for a valid code and HTTP 404 with `{ valid: false }` for an unknown code.
- [ ] AC-8: `GET /admin/sessions/:id/certificate` returns the certificate for any session, not restricted to the caller's own sessions; requires `role IN (examiner, hr_admin, super_admin)`.
- [ ] AC-9: `template_snapshot` captures `company_name`, `logo_base64`, `primary_color`, `signatory_name`, and `signatory_title` from tenant config at the time of first issuance; later branding changes do not alter existing certificates.
- [ ] AC-10: Certificate creation is idempotent: concurrent requests for the same session use an `ON CONFLICT (session_id) DO NOTHING` upsert so only one record is ever created.

## Technical Specification

### Database Schema

```sql
CREATE TABLE certificates (
  id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id         UUID        NOT NULL REFERENCES tenants(id),
  session_id        UUID        NOT NULL UNIQUE REFERENCES exam_sessions(id),
  verification_code UUID        NOT NULL UNIQUE DEFAULT gen_random_uuid(),
  issued_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  employee_name     TEXT        NOT NULL,
  exam_title        TEXT        NOT NULL,
  score_pct         DECIMAL(5,2) NOT NULL,
  template_snapshot JSONB       NOT NULL
);

CREATE INDEX idx_certificates_verification_code ON certificates(verification_code);
CREATE INDEX idx_certificates_tenant_id ON certificates(tenant_id);
```

### API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/portal/sessions/:id/certificate` | any authenticated | Employee downloads own certificate PDF |
| GET | `/api/v1/admin/sessions/:id/certificate` | examiner+ | Admin downloads any session's certificate PDF |
| GET | `/api/v1/verify/:code` | public (no auth) | Verify certificate authenticity by code |

#### Request / Response Shapes

**GET /api/v1/portal/sessions/:id/certificate — Response (200)**
```
Content-Type: application/pdf
Content-Disposition: attachment; filename="certificate-550e8400-e29b-41d4-a716-446655440000.pdf"

<binary PDF bytes>
```

**GET /api/v1/verify/:code — Response (200, valid)**
```json
{
  "data": {
    "valid": true,
    "employee_name": "Aibek Seitkali",
    "exam_title": "Fire Safety Fundamentals",
    "score_pct": 84.50,
    "issued_at": "2026-05-14T10:35:00Z"
  },
  "error": null
}
```

**GET /api/v1/verify/:code — Response (404, invalid)**
```json
{
  "data": { "valid": false },
  "error": null
}
```

**Error — Exam not certifiable (422)**
```json
{
  "data": null,
  "error": { "code": "EXAM_NOT_CERTIFIABLE", "message": "This exam does not issue certificates" }
}
```

**Error — Session not passed (422)**
```json
{
  "data": null,
  "error": { "code": "SESSION_NOT_PASSED", "message": "Certificate is only issued for passed sessions" }
}
```

### `template_snapshot` JSON Structure

```json
{
  "company_name": "Kazakh National Corp",
  "logo_base64": "data:image/png;base64,...",
  "primary_color": "#1A56DB",
  "signatory_name": "Aigerim Bekova",
  "signatory_title": "Head of Human Resources"
}
```

### Certificate Lazy-Creation Logic (Go pseudo-code)

```go
func (s *CertificateService) GetOrCreate(ctx context.Context, sessionID uuid.UUID, callerID uuid.UUID) (*Certificate, error) {
    // 1. Validate session ownership, status, passed, certificate_enabled
    session, exam, err := s.repo.GetSessionWithExam(ctx, sessionID)
    // ... validation guards ...

    // 2. Try to fetch existing certificate
    cert, err := s.repo.GetBySessionID(ctx, sessionID)
    if err == nil {
        return cert, nil // already exists
    }

    // 3. Build snapshot from tenant config
    tenantCfg := s.tenantCache.Get(session.TenantID)
    snapshot := TemplateSnapshot{
        CompanyName:    tenantCfg.CompanyName,
        LogoBase64:     tenantCfg.LogoBase64,
        PrimaryColor:   tenantCfg.PrimaryColor,
        SignatoryName:  tenantCfg.SignatoryName,
        SignatoryTitle: tenantCfg.SignatoryTitle,
    }

    // 4. Insert with ON CONFLICT DO NOTHING
    cert, err = s.repo.CreateCertificate(ctx, CertificateInsert{
        SessionID:        sessionID,
        TenantID:         session.TenantID,
        EmployeeName:     session.EmployeeName,
        ExamTitle:        exam.Title,
        ScorePct:         session.ScorePct,
        TemplateSnapshot: snapshot,
    })
    return cert, err
}
```

## Notes
- No PDF binary is stored in the database; PDFs are regenerated on every request from `certificates` record data. This avoids large binary blobs in PostgreSQL while keeping the source of truth consistent.
- The `/verify/:code` endpoint must not be rate-limited to zero but should apply a generous rate limit (e.g. 60 req/min per IP) to prevent enumeration attacks.
- The `verification_code` UUID is generated by PostgreSQL (`DEFAULT gen_random_uuid()`), not by application code, to ensure uniqueness even under concurrent inserts.
