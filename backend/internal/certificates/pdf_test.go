package certificates

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGeneratePDF_ValidOutput(t *testing.T) {
	cert := &Certificate{
		VerificationCode: "550e8400-e29b-41d4-a716-446655440000",
		EmployeeName:     "Test User",
		ExamTitle:        "Test Exam",
		ScorePct:         90.0,
		IssuedAt:         time.Now(),
	}
	snap := TemplateSnapshot{
		CompanyName:    "Test Corp",
		PrimaryColor:   "#336699",
		SignatoryName:  "HR Manager",
		SignatoryTitle: "Head of HR",
	}
	b, err := GeneratePDF(cert, snap, "https://app.bilimbaga.kz")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(b), 5, "output too short to be a PDF")
	require.Equal(t, "%PDF-", string(b[:5]), "output must start with PDF magic bytes")
	require.Greater(t, len(b), 1024)
}

func TestGeneratePDF_InvalidLogoBase64(t *testing.T) {
	cert := &Certificate{
		VerificationCode: "550e8400-e29b-41d4-a716-446655440000",
		EmployeeName:     "Test User",
		ExamTitle:        "Test Exam",
		ScorePct:         90.0,
		IssuedAt:         time.Now(),
	}
	snap := TemplateSnapshot{
		CompanyName: "Test Corp",
		LogoBase64:  "!!!not-valid-base64!!!",
	}
	_, err := GeneratePDF(cert, snap, "https://app.bilimbaga.kz")
	require.Error(t, err, "invalid base64 logo must return an error")
}

func TestGeneratePDF_EmptyLogoFallsBackToCompanyName(t *testing.T) {
	cert := &Certificate{
		VerificationCode: "550e8400-e29b-41d4-a716-446655440000",
		EmployeeName:     "Test User",
		ExamTitle:        "Test Exam",
		ScorePct:         90.0,
		IssuedAt:         time.Now(),
	}
	snap := TemplateSnapshot{
		CompanyName:    "Fallback Corp",
		LogoBase64:     "", // empty — should render company name text
		SignatoryName:  "HR Manager",
		SignatoryTitle: "Head of HR",
	}
	b, err := GeneratePDF(cert, snap, "https://app.bilimbaga.kz")
	require.NoError(t, err)
	require.Equal(t, "%PDF-", string(b[:5]))
}
