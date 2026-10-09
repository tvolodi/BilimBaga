package certificates

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// TenantConfigProvider exposes the raw tenant configuration cache.
// Implemented by *tenant.service (via tenant.Service interface extension).
type TenantConfigProvider interface {
	GetAllConfig() map[string]json.RawMessage
}

// Service defines the business logic for the certificates domain.
type Service interface {
	// GetOrCreate returns the existing certificate for the session or creates one on first call.
	// If isAdmin is false, callerID must match the session owner or ErrNotOwner is returned.
	GetOrCreate(ctx context.Context, sessionID, callerID string, isAdmin bool) (*Certificate, TemplateSnapshot, error)

	// GetByVerificationCode looks up a certificate by its public verification code.
	// Always returns a VerifyResponse (never an error for unknown codes — valid=false is used).
	GetByVerificationCode(ctx context.Context, code string) (*VerifyResponse, error)
}

type service struct {
	repo      Repository
	tenantCfg TenantConfigProvider
}

// NewService creates a new certificates Service.
func NewService(repo Repository, tenantCfg TenantConfigProvider) Service {
	return &service{repo: repo, tenantCfg: tenantCfg}
}

func (s *service) GetOrCreate(ctx context.Context, sessionID, callerID string, isAdmin bool) (*Certificate, TemplateSnapshot, error) {
	row, err := s.repo.GetSessionForCertificate(ctx, sessionID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, TemplateSnapshot{}, ErrNotFound
		}
		return nil, TemplateSnapshot{}, fmt.Errorf("certificates: GetOrCreate: fetch session: %w", err)
	}

	// AC-1: ownership check (skipped for admin callers).
	if !isAdmin && row.UserID != callerID {
		return nil, TemplateSnapshot{}, ErrNotOwner
	}

	// AC-4: session must be in submitted state.
	if row.Status != "submitted" {
		return nil, TemplateSnapshot{}, ErrNotSubmitted
	}

	// AC-2: exam must have certificate_enabled = true.
	if !row.CertificateEnabled {
		return nil, TemplateSnapshot{}, ErrNotCertifiable
	}

	// AC-3: session must have passed.
	if !row.Passed {
		return nil, TemplateSnapshot{}, ErrNotPassed
	}

	// AC-5: return existing certificate if already issued.
	cert, err := s.repo.GetBySessionID(ctx, sessionID)
	if err == nil {
		snap, snapErr := parseSnapshot(cert.TemplateSnapshot)
		if snapErr != nil {
			snap = TemplateSnapshot{}
		}
		return cert, snap, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, TemplateSnapshot{}, fmt.Errorf("certificates: GetOrCreate: check existing: %w", err)
	}

	// AC-9: capture tenant branding at issuance time.
	snap := s.buildSnapshot()

	scorePct := 0.0
	if row.ScorePct != nil {
		scorePct = *row.ScorePct
	}

	// AC-10: ON CONFLICT (session_id) DO NOTHING makes this idempotent under concurrency.
	cert, err = s.repo.Create(ctx, certInsert{
		SessionID:        sessionID,
		EmployeeName:     row.EmployeeName,
		ExamTitle:        row.ExamTitle,
		ScorePct:         scorePct,
		TemplateSnapshot: snap,
	})
	if err != nil {
		return nil, TemplateSnapshot{}, fmt.Errorf("certificates: GetOrCreate: create: %w", err)
	}

	return cert, snap, nil
}

func (s *service) GetByVerificationCode(ctx context.Context, code string) (*VerifyResponse, error) {
	// verification_code is a UUID column: a malformed code can never match, and
	// passing it to Postgres would raise an invalid-input-syntax error (-> 500).
	parsed, perr := uuid.Parse(code)
	if perr != nil {
		return &VerifyResponse{Valid: false}, nil
	}
	cert, err := s.repo.GetByVerificationCode(ctx, parsed.String())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// AC-7: unknown code → valid=false, still HTTP 200.
			return &VerifyResponse{Valid: false}, nil
		}
		return nil, fmt.Errorf("certificates: GetByVerificationCode: %w", err)
	}

	issuedAt := cert.IssuedAt
	scorePct := cert.ScorePct
	return &VerifyResponse{
		Valid:        true,
		EmployeeName: cert.EmployeeName,
		ExamTitle:    cert.ExamTitle,
		ScorePct:     &scorePct,
		IssuedAt:     &issuedAt,
	}, nil
}

// buildSnapshot reads tenant config from the cache and maps keys to the TemplateSnapshot fields.
// Missing keys fall back to safe defaults.
func (s *service) buildSnapshot() TemplateSnapshot {
	cfg := s.tenantCfg.GetAllConfig()
	var snap TemplateSnapshot

	jsonString := func(key string) string {
		v, ok := cfg[key]
		if !ok {
			return ""
		}
		var s string
		_ = json.Unmarshal(v, &s)
		return s
	}

	// Prefer company_name; fall back to app_name.
	snap.CompanyName = jsonString("company_name")
	if snap.CompanyName == "" {
		snap.CompanyName = jsonString("app_name")
	}
	snap.LogoBase64 = jsonString("logo")
	snap.PrimaryColor = jsonString("primary_color")
	snap.SignatoryName = jsonString("signatory_name")
	snap.SignatoryTitle = jsonString("signatory_title")

	return snap
}

func parseSnapshot(raw json.RawMessage) (TemplateSnapshot, error) {
	var snap TemplateSnapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		return TemplateSnapshot{}, fmt.Errorf("certificates: parseSnapshot: %w", err)
	}
	return snap, nil
}
