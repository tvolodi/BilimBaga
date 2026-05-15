package certificates

import (
	"encoding/json"
	"errors"
	"time"
)

// Sentinel errors for the certificates domain.
var (
	ErrNotOwner       = errors.New("session does not belong to caller")
	ErrNotCertifiable = errors.New("exam does not issue certificates")
	ErrNotPassed      = errors.New("session was not passed")
	ErrNotSubmitted   = errors.New("session is not submitted")
	ErrNotFound       = errors.New("certificate not found")
)

// TemplateSnapshot captures tenant branding at the time of first certificate issuance.
// Stored as JSONB so later branding changes do not alter existing certificates.
type TemplateSnapshot struct {
	CompanyName    string `json:"company_name"`
	LogoBase64     string `json:"logo_base64"`
	PrimaryColor   string `json:"primary_color"`
	SignatoryName  string `json:"signatory_name"`
	SignatoryTitle string `json:"signatory_title"`
}

// Certificate is the persisted certificate record.
type Certificate struct {
	ID               string          `db:"id"`
	SessionID        string          `db:"session_id"`
	VerificationCode string          `db:"verification_code"`
	IssuedAt         time.Time       `db:"issued_at"`
	EmployeeName     string          `db:"employee_name"`
	ExamTitle        string          `db:"exam_title"`
	ScorePct         float64         `db:"score_pct"`
	TemplateSnapshot json.RawMessage `db:"template_snapshot"`
}

// VerifyResponse is the payload for the public verify endpoint.
// When Valid is false all other fields are omitted (omitempty on pointer/string fields).
type VerifyResponse struct {
	Valid        bool       `json:"valid"`
	EmployeeName string     `json:"employee_name,omitempty"`
	ExamTitle    string     `json:"exam_title,omitempty"`
	ScorePct     *float64   `json:"score_pct,omitempty"`
	IssuedAt     *time.Time `json:"issued_at,omitempty"`
}
