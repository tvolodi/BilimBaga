package tenant

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/bilimbaga/bilimbaga/internal/upload"
)

const maxLogoBytes = upload.MaxLogoBytes // 2 MB (AC-3: FR-BB64)

// publicKeys is the set of keys exposed via GET /api/v1/tenant/config.
var publicKeys = map[string]struct{}{
	"app_name":          {},
	"primary_color":     {},
	"accent_color":      {},
	"default_locale":    {},
	"available_locales": {},
}

// ValidationError represents a domain validation failure.
type ValidationError struct {
	Code    string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Service defines the business-logic interface for tenant configuration.
type Service interface {
	LoadCache(ctx context.Context) error
	InvalidateAndRefresh(ctx context.Context) error
	GetPublicConfig() map[string]json.RawMessage
	// GetAllConfig returns the full raw configuration cache (all keys, including non-public ones).
	// Used by the certificates package to capture a branding snapshot at issuance time.
	GetAllConfig() map[string]json.RawMessage
	GetLogoData() ([]byte, string, error)
	UpdateConfig(ctx context.Context, updates map[string]json.RawMessage) ([]string, error)

	// GetDefaultLocale returns the configured default_locale, or empty string if unset.
	GetDefaultLocale() string
	// GetAvailableLocales returns the configured available_locales list, or nil if unset.
	GetAvailableLocales() []string
}

type service struct {
	repo  Repository
	mu    sync.RWMutex
	cache map[string]json.RawMessage
}

// NewService creates a new Service backed by the given Repository.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// LoadCache loads all rows from the repository into the in-memory cache.
// It must be called once at startup before the HTTP server begins accepting requests.
func (s *service) LoadCache(ctx context.Context) error {
	data, err := s.repo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("tenant.LoadCache: %w", err)
	}
	s.mu.Lock()
	s.cache = data
	s.mu.Unlock()
	return nil
}

// InvalidateAndRefresh clears and reloads the in-memory cache from the repository.
func (s *service) InvalidateAndRefresh(ctx context.Context) error {
	data, err := s.repo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("tenant.InvalidateAndRefresh: %w", err)
	}
	s.mu.Lock()
	s.cache = data
	s.mu.Unlock()
	return nil
}

// GetPublicConfig returns only the public-facing subset of the tenant config
// (excludes the logo key).
func (s *service) GetPublicConfig() map[string]json.RawMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]json.RawMessage, len(publicKeys))
	for k := range publicKeys {
		if v, ok := s.cache[k]; ok {
			result[k] = v
		}
	}
	return result
}

// GetAllConfig returns the full raw configuration cache including non-public keys.
// The returned map is a shallow copy — callers must not modify it.
func (s *service) GetAllConfig() map[string]json.RawMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]json.RawMessage, len(s.cache))
	for k, v := range s.cache {
		result[k] = v
	}
	return result
}

// GetLogoData decodes the base64 data-URI logo stored in the cache.
// Returns (nil, "", nil) when no logo is configured (value is JSON null or empty).
func (s *service) GetLogoData() ([]byte, string, error) {
	s.mu.RLock()
	raw, ok := s.cache["logo"]
	s.mu.RUnlock()
	if !ok {
		return nil, "", nil
	}

	// The stored value is a JSON-encoded string (or JSON null).
	var logoStr *string
	if err := json.Unmarshal(raw, &logoStr); err != nil {
		return nil, "", fmt.Errorf("tenant.GetLogoData: unmarshal: %w", err)
	}
	if logoStr == nil || *logoStr == "" {
		return nil, "", nil
	}

	// Expected format: data:<mime>;base64,<b64data>
	parts := strings.SplitN(*logoStr, ",", 2)
	if len(parts) != 2 {
		return nil, "", fmt.Errorf("tenant.GetLogoData: invalid data URI format")
	}
	// parts[0] is e.g. "data:image/png;base64"
	metaParts := strings.SplitN(parts[0], ":", 2)
	if len(metaParts) != 2 {
		return nil, "", fmt.Errorf("tenant.GetLogoData: invalid data URI meta")
	}
	mimeType := strings.SplitN(metaParts[1], ";", 2)[0]

	decoded, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, "", fmt.Errorf("tenant.GetLogoData: base64 decode: %w", err)
	}
	return decoded, mimeType, nil
}

// UpdateConfig validates and applies a partial update to tenant configuration.
// Returns the list of keys that were written.
func (s *service) UpdateConfig(ctx context.Context, updates map[string]json.RawMessage) ([]string, error) {
	if err := s.validateLocaleConstraint(updates); err != nil {
		return nil, err
	}

	if primaryRaw, ok := updates["primary_color"]; ok {
		if err := validatePrimaryColor(primaryRaw); err != nil {
			return nil, err
		}
	}

	if logoRaw, ok := updates["logo"]; ok {
		var logoStr *string
		if err := json.Unmarshal(logoRaw, &logoStr); err != nil {
			return nil, fmt.Errorf("tenant.UpdateConfig: invalid logo value: %w", err)
		}
		if logoStr != nil && *logoStr != "" {
			parts := strings.SplitN(*logoStr, ",", 2)
			if len(parts) == 2 {
				decoded, err := base64.StdEncoding.DecodeString(parts[1])
				if err != nil {
					return nil, fmt.Errorf("tenant.UpdateConfig: invalid logo base64: %w", err)
				}
				// AC-3 (FR-BB64): validate size and magic bytes server-side.
				if err := upload.ValidateLogoFile(decoded); err != nil {
					switch err {
					case upload.ErrFileTooLarge:
						return nil, &ValidationError{Code: "LOGO_TOO_LARGE", Message: "logo exceeds 2 MB limit"}
					default:
						return nil, &ValidationError{Code: "INVALID_LOGO_TYPE", Message: "logo must be a PNG or JPEG image"}
					}
				}
			}
		}
	}

	var updatedKeys []string
	for key, value := range updates {
		if err := s.repo.Upsert(ctx, key, value); err != nil {
			return nil, fmt.Errorf("tenant.UpdateConfig: %w", err)
		}
		updatedKeys = append(updatedKeys, key)
	}

	if err := s.InvalidateAndRefresh(ctx); err != nil {
		return nil, fmt.Errorf("tenant.UpdateConfig: refresh: %w", err)
	}

	return updatedKeys, nil
}

// GetDefaultLocale returns the configured default_locale, or empty string if unset.
func (s *service) GetDefaultLocale() string {
	s.mu.RLock()
	raw, ok := s.cache["default_locale"]
	s.mu.RUnlock()
	if !ok {
		return ""
	}
	var v string
	if err := json.Unmarshal(raw, &v); err != nil {
		return ""
	}
	return v
}

// GetAvailableLocales returns the configured available_locales list, or nil if unset.
func (s *service) GetAvailableLocales() []string {
	s.mu.RLock()
	raw, ok := s.cache["available_locales"]
	s.mu.RUnlock()
	if !ok {
		return nil
	}
	var v []string
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil
	}
	return v
}

// validatePrimaryColor enforces FR-BB320 AC-1: the primary colour must be a #rgb or #rrggbb
// value whose contrast against white reaches 4.5:1, because it carries the white text on
// primary surfaces. A value that cannot be parsed has no contrast ratio and is rejected too.
func validatePrimaryColor(raw json.RawMessage) error {
	var colour string
	if err := json.Unmarshal(raw, &colour); err != nil {
		return &ValidationError{Code: "VALIDATION_ERROR", Message: "primary_color must be a hex colour string"}
	}
	ratio, err := ContrastRatio(colour, "#ffffff")
	if err != nil {
		return &ValidationError{Code: "VALIDATION_ERROR", Message: "primary_color must be a #rgb or #rrggbb hex colour"}
	}
	if ratio < minPrimaryContrast {
		return &ValidationError{
			Code: "VALIDATION_ERROR",
			Message: fmt.Sprintf("primary_color contrast against white is %.2f:1; it must be at least %.1f:1",
				ratio, minPrimaryContrast),
		}
	}
	return nil
}

// validateLocaleConstraint enforces that default_locale ∈ available_locales.
func (s *service) validateLocaleConstraint(updates map[string]json.RawMessage) error {
	s.mu.RLock()
	currentDefault := s.cache["default_locale"]
	currentAvailable := s.cache["available_locales"]
	s.mu.RUnlock()

	newDefault := currentDefault
	if v, ok := updates["default_locale"]; ok {
		newDefault = v
	}
	newAvailable := currentAvailable
	if v, ok := updates["available_locales"]; ok {
		newAvailable = v
	}

	if newDefault == nil || newAvailable == nil {
		return nil
	}

	var defaultLocale string
	if err := json.Unmarshal(newDefault, &defaultLocale); err != nil {
		return &ValidationError{Code: "INVALID_LOCALE", Message: "default_locale must be a string"}
	}

	var availableLocales []string
	if err := json.Unmarshal(newAvailable, &availableLocales); err != nil {
		return &ValidationError{Code: "INVALID_LOCALE", Message: "available_locales must be an array of strings"}
	}

	for _, loc := range availableLocales {
		if loc == defaultLocale {
			return nil
		}
	}
	return &ValidationError{Code: "INVALID_LOCALE", Message: "default_locale must be in available_locales"}
}
