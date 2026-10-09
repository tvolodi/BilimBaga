package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

const (
	// BootstrapAdminEmail is the account seeded by migration 029.
	BootstrapAdminEmail = "admin@bilimbaga.local"
	// DefaultAdminPassword is the publicly documented seed password (migrations 029/030).
	DefaultAdminPassword = "Admin1234!"
)

// BootstrapStore is the persistence the admin bootstrap needs. It is separate from
// Repository so the large auth fakes do not have to implement it.
type BootstrapStore interface {
	// GetAdminCredentials returns the seeded admin's id and password hash, or ErrNotFound.
	GetAdminCredentials(ctx context.Context, email string) (id, passwordHash string, err error)
	// SetAdminPassword stores newHash and sets force_password_change = forceChange, only if
	// the stored hash still equals oldHash. It reports whether a row was updated.
	SetAdminPassword(ctx context.Context, id, oldHash, newHash string, forceChange bool) (bool, error)
	// RequireAdminPasswordChange sets force_password_change = true, only if the stored
	// hash still equals hash. It reports whether a row matched.
	RequireAdminPasswordChange(ctx context.Context, id, hash string) (bool, error)
}

// BootstrapOutcome describes what BootstrapAdmin did.
type BootstrapOutcome string

const (
	BootstrapNoAdmin           BootstrapOutcome = "no_admin"           // admin account not present; nothing done
	BootstrapNotDefault        BootstrapOutcome = "not_default"        // admin already has its own password; untouched
	BootstrapPasswordApplied   BootstrapOutcome = "password_applied"   // env password hashed and stored
	BootstrapPasswordGenerated BootstrapOutcome = "password_generated" // random one-time password stored; force change set
	BootstrapForcedChange      BootstrapOutcome = "forced_change"      // default password kept, change forced
	BootstrapRaced             BootstrapOutcome = "concurrent_update"  // another writer changed the row first
)

// BootstrapOptions configures BootstrapAdmin.
type BootstrapOptions struct {
	Password string // BOOTSTRAP_ADMIN_PASSWORD; empty = unset
	Generate bool   // generate a random one-time password when Password is empty
	Cost     int    // bcrypt cost
}

// BootstrapResult is the outcome plus, for BootstrapPasswordGenerated only, the generated
// password. The caller must log it at most once and never persist it.
type BootstrapResult struct {
	Outcome           BootstrapOutcome
	GeneratedPassword string
	// StillDefault is true when the admin still has the default password after the call
	// (callers log a WARNING).
	StillDefault bool
	// EnvPasswordInvalid is true when BOOTSTRAP_ADMIN_PASSWORD was set but invalid and was
	// ignored because the admin is absent or already rotated (callers log a WARNING).
	EnvPasswordInvalid bool
}

// BootstrapAdmin makes the seeded super_admin safe at startup (ISS-150/ISS-152).
//
//   - Admin missing or no longer on the default password: nothing is changed (a configured
//     password is never applied over a password someone already chose).
//   - Default password + opts.Password: validated (ValidateComplexity, must differ from the
//     default), bcrypt-hashed, stored; force_password_change cleared (operator chose it).
//   - Default password + opts.Generate: random one-time password stored with
//     force_password_change = true and returned for one-time logging.
//   - Otherwise force_password_change is set so the default credential must be changed at
//     first login; StillDefault is true so the caller can warn.
//
// The env password is never logged or included in errors.
func BootstrapAdmin(ctx context.Context, store BootstrapStore, opts BootstrapOptions) (BootstrapResult, error) {
	// The env password is validated up front but only enforced where it matters: if the
	// admin already rotated away from the default, an invalid value is ignored with a warning.
	envErr := validateBootstrapPassword(opts.Password)

	id, hash, err := store.GetAdminCredentials(ctx, BootstrapAdminEmail)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return BootstrapResult{Outcome: BootstrapNoAdmin, EnvPasswordInvalid: envErr != nil}, nil
		}
		return BootstrapResult{}, fmt.Errorf("auth.BootstrapAdmin: load admin: %w", err)
	}

	if !HasDefaultAdminPassword(hash) {
		return BootstrapResult{Outcome: BootstrapNotDefault, EnvPasswordInvalid: envErr != nil}, nil
	}
	if envErr != nil {
		return BootstrapResult{}, envErr
	}

	password, force, outcome := opts.Password, false, BootstrapPasswordApplied
	if password == "" && opts.Generate {
		password, err = generatePassword()
		if err != nil {
			return BootstrapResult{}, fmt.Errorf("auth.BootstrapAdmin: generate password: %w", err)
		}
		force, outcome = true, BootstrapPasswordGenerated
	}

	if password != "" {
		newHash, err := bcrypt.GenerateFromPassword([]byte(password), opts.Cost)
		if err != nil {
			return BootstrapResult{}, fmt.Errorf("auth.BootstrapAdmin: hash password: %w", err)
		}
		ok, err := store.SetAdminPassword(ctx, id, hash, string(newHash), force)
		if err != nil {
			return BootstrapResult{}, fmt.Errorf("auth.BootstrapAdmin: set password: %w", err)
		}
		if !ok {
			return BootstrapResult{Outcome: BootstrapRaced}, nil
		}
		res := BootstrapResult{Outcome: outcome}
		if outcome == BootstrapPasswordGenerated {
			res.GeneratedPassword = password
		}
		return res, nil
	}

	ok, err := store.RequireAdminPasswordChange(ctx, id, hash)
	if err != nil {
		return BootstrapResult{}, fmt.Errorf("auth.BootstrapAdmin: force password change: %w", err)
	}
	if !ok {
		return BootstrapResult{Outcome: BootstrapRaced, StillDefault: true}, nil
	}
	return BootstrapResult{Outcome: BootstrapForcedChange, StillDefault: true}, nil
}

// validateBootstrapPassword checks the env password; "" (unset) is valid. Errors never
// include the value.
func validateBootstrapPassword(pw string) error {
	if pw == "" {
		return nil
	}
	if len(pw) > maxBcryptBytes { // bytes, not runes: bcrypt truncates/rejects beyond 72 bytes
		return errors.New("auth.BootstrapAdmin: BOOTSTRAP_ADMIN_PASSWORD is longer than 72 bytes (bcrypt limit)")
	}
	if err := ValidateComplexity(pw); err != nil {
		return errors.New("auth.BootstrapAdmin: BOOTSTRAP_ADMIN_PASSWORD does not meet the password complexity policy (8+ chars, upper, lower, digit)")
	}
	if pw == DefaultAdminPassword {
		return errors.New("auth.BootstrapAdmin: BOOTSTRAP_ADMIN_PASSWORD must differ from the default admin password")
	}
	return nil
}

// HasDefaultAdminPassword reports whether hash is a bcrypt hash of the default admin password.
func HasDefaultAdminPassword(hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(DefaultAdminPassword)) == nil
}

// maxBcryptBytes is bcrypt's maximum input length.
const maxBcryptBytes = 72

const (
	pwLower  = "abcdefghijkmnopqrstuvwxyz"
	pwUpper  = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	pwDigit  = "23456789"
	pwAll    = pwLower + pwUpper + pwDigit
	pwLength = 20 // must stay < maxBcryptBytes
)

var _ = [maxBcryptBytes - pwLength]struct{}{} // compile-time: pwLength <= maxBcryptBytes

// generatePassword returns a random password satisfying ValidateComplexity.
func generatePassword() (string, error) {
	pick := func(set string) (byte, error) {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(set))))
		if err != nil {
			return 0, err
		}
		return set[n.Int64()], nil
	}
	classes := []string{pwLower, pwUpper, pwDigit}
	out := make([]byte, pwLength)
	for i := range out {
		set := pwAll
		if i < len(classes) { // guarantee one of each class, then shuffle below
			set = classes[i]
		}
		c, err := pick(set)
		if err != nil {
			return "", err
		}
		out[i] = c
	}
	for i := len(out) - 1; i > 0; i-- {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return "", err
		}
		j := n.Int64()
		out[i], out[j] = out[j], out[i]
	}
	return string(out), nil
}

type pgBootstrapStore struct{ db *sqlx.DB }

// NewBootstrapStore creates the PostgreSQL-backed BootstrapStore.
func NewBootstrapStore(db *sqlx.DB) BootstrapStore { return &pgBootstrapStore{db: db} }

func (s *pgBootstrapStore) GetAdminCredentials(ctx context.Context, email string) (string, string, error) {
	const q = `SELECT id, password_hash FROM users WHERE lower(email) = lower($1)`
	var row struct {
		ID           string `db:"id"`
		PasswordHash string `db:"password_hash"`
	}
	if err := s.db.GetContext(ctx, &row, q, email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", ErrNotFound
		}
		return "", "", fmt.Errorf("auth.GetAdminCredentials: %w", err)
	}
	return row.ID, row.PasswordHash, nil
}

func (s *pgBootstrapStore) SetAdminPassword(ctx context.Context, id, oldHash, newHash string, forceChange bool) (bool, error) {
	// password_changed_at is stamped so access tokens issued under the default password
	// are rejected by the password-epoch check (ISS-105).
	const q = `
		UPDATE users
		SET    password_hash = $1, force_password_change = $4,
		       failed_attempts = 0, locked_until = NULL,
		       password_changed_at = now(), updated_at = now()
		WHERE  id = $2 AND password_hash = $3`
	res, err := s.db.ExecContext(ctx, q, newHash, id, oldHash, forceChange)
	if err != nil {
		return false, fmt.Errorf("auth.SetAdminPassword: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("auth.SetAdminPassword: rows affected: %w", err)
	}
	return n > 0, nil
}

func (s *pgBootstrapStore) RequireAdminPasswordChange(ctx context.Context, id, hash string) (bool, error) {
	const q = `
		UPDATE users
		SET    force_password_change = true, updated_at = now()
		WHERE  id = $1 AND password_hash = $2`
	res, err := s.db.ExecContext(ctx, q, id, hash)
	if err != nil {
		return false, fmt.Errorf("auth.RequireAdminPasswordChange: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("auth.RequireAdminPasswordChange: rows affected: %w", err)
	}
	return n > 0, nil
}
