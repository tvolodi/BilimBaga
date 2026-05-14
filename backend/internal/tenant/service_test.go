package tenant

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRepository is an in-memory Repository used in unit tests.
type mockRepository struct {
	data map[string]json.RawMessage
}

func newMockRepository(data map[string]json.RawMessage) Repository {
	return &mockRepository{data: data}
}

func (m *mockRepository) GetAll(_ context.Context) (map[string]json.RawMessage, error) {
	copy := make(map[string]json.RawMessage, len(m.data))
	for k, v := range m.data {
		copy[k] = v
	}
	return copy, nil
}

func (m *mockRepository) Upsert(_ context.Context, key string, value json.RawMessage) error {
	if m.data == nil {
		m.data = make(map[string]json.RawMessage)
	}
	m.data[key] = value
	return nil
}

// defaultSeedData returns a seed map matching the migration defaults.
func defaultSeedData() map[string]json.RawMessage {
	return map[string]json.RawMessage{
		"app_name":          json.RawMessage(`"BilimBaga"`),
		"logo":              json.RawMessage(`null`),
		"primary_color":     json.RawMessage(`"#0ea5e9"`),
		"accent_color":      json.RawMessage(`"#f59e0b"`),
		"default_locale":    json.RawMessage(`"kk"`),
		"available_locales": json.RawMessage(`["kk","ru","en"]`),
	}
}

// TestLoadCache verifies that LoadCache populates the in-memory cache.
func TestLoadCache(t *testing.T) {
	repo := newMockRepository(defaultSeedData())
	svc := NewService(repo)

	err := svc.LoadCache(context.Background())
	require.NoError(t, err)

	cfg := svc.GetPublicConfig()
	assert.NotEmpty(t, cfg)
	assert.Contains(t, cfg, "app_name")
	assert.Contains(t, cfg, "primary_color")
	assert.Contains(t, cfg, "accent_color")
	assert.Contains(t, cfg, "default_locale")
	assert.Contains(t, cfg, "available_locales")
}

// TestGetPublicConfig verifies that GetPublicConfig excludes the logo key.
func TestGetPublicConfig(t *testing.T) {
	repo := newMockRepository(defaultSeedData())
	svc := NewService(repo)

	require.NoError(t, svc.LoadCache(context.Background()))

	cfg := svc.GetPublicConfig()
	assert.NotContains(t, cfg, "logo", "logo must not appear in public config")
	assert.Len(t, cfg, 5)
}

// TestUpdateConfig_LocaleConstraint verifies that default_locale must be in available_locales.
func TestUpdateConfig_LocaleConstraint(t *testing.T) {
	tests := []struct {
		name        string
		updates     map[string]json.RawMessage
		wantErrCode string
	}{
		{
			name: "invalid: default_locale not in available_locales",
			updates: map[string]json.RawMessage{
				"default_locale":    json.RawMessage(`"fr"`),
				"available_locales": json.RawMessage(`["kk","ru","en"]`),
			},
			wantErrCode: "INVALID_LOCALE",
		},
		{
			name: "valid: default_locale present in available_locales",
			updates: map[string]json.RawMessage{
				"default_locale":    json.RawMessage(`"ru"`),
				"available_locales": json.RawMessage(`["kk","ru","en"]`),
			},
			wantErrCode: "",
		},
		{
			name: "valid: only updating default_locale that already exists in current available_locales",
			updates: map[string]json.RawMessage{
				"default_locale": json.RawMessage(`"ru"`),
			},
			wantErrCode: "",
		},
		{
			name: "invalid: updating available_locales to exclude current default_locale",
			updates: map[string]json.RawMessage{
				"available_locales": json.RawMessage(`["ru","en"]`),
			},
			wantErrCode: "INVALID_LOCALE",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newMockRepository(defaultSeedData())
			svc := NewService(repo)
			require.NoError(t, svc.LoadCache(context.Background()))

			_, err := svc.UpdateConfig(context.Background(), tc.updates)
			if tc.wantErrCode != "" {
				require.Error(t, err)
				var valErr *ValidationError
				require.ErrorAs(t, err, &valErr)
				assert.Equal(t, tc.wantErrCode, valErr.Code)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestUpdateConfig_LogoSizeLimit verifies that logos exceeding 1 MB are rejected.
func TestUpdateConfig_LogoSizeLimit(t *testing.T) {
	// Build a base64 string whose decoded length exceeds maxLogoBytes.
	// Each 4 base64 chars decode to 3 bytes; we need > 1 MB decoded bytes.
	b64Payload := "data:image/png;base64," + strings.Repeat("AAAA", (maxLogoBytes/3)+1)
	logoVal, _ := json.Marshal(b64Payload)

	repo := newMockRepository(defaultSeedData())
	svc := NewService(repo)
	require.NoError(t, svc.LoadCache(context.Background()))

	_, err := svc.UpdateConfig(context.Background(), map[string]json.RawMessage{
		"logo": logoVal,
	})
	require.Error(t, err)
	var valErr *ValidationError
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "LOGO_TOO_LARGE", valErr.Code)
}

// TestInvalidateAndRefresh verifies that the cache is updated after a refresh.
func TestInvalidateAndRefresh(t *testing.T) {
	seed := defaultSeedData()
	repo := newMockRepository(seed)
	svc := NewService(repo)
	require.NoError(t, svc.LoadCache(context.Background()))

	// Simulate an external update to the repository.
	repo.(*mockRepository).data["app_name"] = json.RawMessage(`"Updated Name"`)

	require.NoError(t, svc.InvalidateAndRefresh(context.Background()))

	cfg := svc.GetPublicConfig()
	assert.Equal(t, json.RawMessage(`"Updated Name"`), cfg["app_name"])
}
