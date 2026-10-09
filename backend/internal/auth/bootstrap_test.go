package auth

import (
	"context"
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

type fakeBootstrapStore struct {
	exists   bool
	hash     string
	force    bool
	getErr   error
	raceOnce bool
	setCalls int
}

func (f *fakeBootstrapStore) GetAdminCredentials(_ context.Context, _ string) (string, string, error) {
	if f.getErr != nil {
		return "", "", f.getErr
	}
	if !f.exists {
		return "", "", ErrNotFound
	}
	return "admin-id", f.hash, nil
}

func (f *fakeBootstrapStore) SetAdminPassword(_ context.Context, _, oldHash, newHash string, force bool) (bool, error) {
	f.setCalls++
	if f.raceOnce || oldHash != f.hash {
		return false, nil
	}
	f.hash, f.force = newHash, force
	return true, nil
}

func (f *fakeBootstrapStore) RequireAdminPasswordChange(_ context.Context, _, hash string) (bool, error) {
	if f.raceOnce || hash != f.hash {
		return false, nil
	}
	f.force = true
	return true, nil
}

func hashOf(t *testing.T, pw string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	return string(h)
}

func bopts(pw string, gen bool) BootstrapOptions {
	return BootstrapOptions{Password: pw, Generate: gen, Cost: bcrypt.MinCost}
}

func TestBootstrapAdmin_NoAdmin(t *testing.T) {
	s := &fakeBootstrapStore{}
	r, err := BootstrapAdmin(context.Background(), s, bopts("Str0ngInitial!", false))
	if err != nil || r.Outcome != BootstrapNoAdmin {
		t.Fatalf("got %v %v", r, err)
	}
}

func TestBootstrapAdmin_StoreError(t *testing.T) {
	s := &fakeBootstrapStore{getErr: errors.New("db down")}
	if _, err := BootstrapAdmin(context.Background(), s, bopts("", false)); err == nil {
		t.Fatal("expected error")
	}
}

func TestBootstrapAdmin_RotatedPasswordUntouched(t *testing.T) {
	h := hashOf(t, "Rotated2024!")
	s := &fakeBootstrapStore{exists: true, hash: h}
	r, err := BootstrapAdmin(context.Background(), s, bopts("Str0ngInitial!", true))
	if err != nil || r.Outcome != BootstrapNotDefault || r.StillDefault {
		t.Fatalf("got %v %v", r, err)
	}
	if s.hash != h || s.force || s.setCalls != 0 {
		t.Fatal("rotated admin must not be modified")
	}
}

func TestBootstrapAdmin_EnvPasswordApplied(t *testing.T) {
	s := &fakeBootstrapStore{exists: true, hash: hashOf(t, DefaultAdminPassword)}
	r, err := BootstrapAdmin(context.Background(), s, bopts("Str0ngInitial!", false))
	if err != nil || r.Outcome != BootstrapPasswordApplied || r.StillDefault || r.GeneratedPassword != "" {
		t.Fatalf("got %v %v", r, err)
	}
	if bcrypt.CompareHashAndPassword([]byte(s.hash), []byte("Str0ngInitial!")) != nil {
		t.Fatal("env password not stored hashed")
	}
	if HasDefaultAdminPassword(s.hash) || s.force {
		t.Fatal("default must be gone and force cleared")
	}
}

func TestBootstrapAdmin_WeakOrDefaultEnvPasswordRejected(t *testing.T) {
	for _, pw := range []string{"short", "alllowercase1", DefaultAdminPassword} {
		s := &fakeBootstrapStore{exists: true, hash: hashOf(t, DefaultAdminPassword)}
		_, err := BootstrapAdmin(context.Background(), s, bopts(pw, false))
		if err == nil {
			t.Fatalf("%q should be rejected", pw)
		}
		if strings.Contains(err.Error(), pw) {
			t.Fatalf("error leaks the password: %v", err)
		}
		if s.setCalls != 0 {
			t.Fatal("store must not be touched")
		}
	}
}

func TestBootstrapAdmin_GeneratedPassword(t *testing.T) {
	s := &fakeBootstrapStore{exists: true, hash: hashOf(t, DefaultAdminPassword)}
	r, err := BootstrapAdmin(context.Background(), s, bopts("", true))
	if err != nil || r.Outcome != BootstrapPasswordGenerated {
		t.Fatalf("got %v %v", r, err)
	}
	if ValidateComplexity(r.GeneratedPassword) != nil {
		t.Fatal("generated password must satisfy the policy")
	}
	if bcrypt.CompareHashAndPassword([]byte(s.hash), []byte(r.GeneratedPassword)) != nil || !s.force {
		t.Fatal("generated password must be stored hashed with force change")
	}
	p2, _ := generatePassword()
	if p2 == r.GeneratedPassword {
		t.Fatal("passwords must be random")
	}
}

func TestBootstrapAdmin_NothingConfiguredForcesChangeAndWarns(t *testing.T) {
	s := &fakeBootstrapStore{exists: true, hash: hashOf(t, DefaultAdminPassword)}
	r, err := BootstrapAdmin(context.Background(), s, bopts("", false))
	if err != nil || r.Outcome != BootstrapForcedChange || !r.StillDefault {
		t.Fatalf("got %v %v", r, err)
	}
	if !s.force || !HasDefaultAdminPassword(s.hash) {
		t.Fatal("default password kept but change forced")
	}
}

func TestBootstrapAdmin_Race(t *testing.T) {
	s := &fakeBootstrapStore{exists: true, hash: hashOf(t, DefaultAdminPassword), raceOnce: true}
	r, err := BootstrapAdmin(context.Background(), s, bopts("Str0ngInitial!", false))
	if err != nil || r.Outcome != BootstrapRaced {
		t.Fatalf("got %v %v", r, err)
	}
}

func TestHasDefaultAdminPassword_MigrationHashes(t *testing.T) {
	// Hash literals shipped in migrations 029 and 030.
	for _, h := range []string{
		"$2b$12$.b2VsbiluRqUkJuT4gpfPOGpE4TLD3RTNZtXjHFUcqrIgFNVxyKj2",
		"$2a$12$v7BCGG1mNmqtooAWqav4LuoB5i4n5hWA06iMyYLT.jPRMA1B9Mbo6",
	} {
		if !HasDefaultAdminPassword(h) {
			t.Errorf("%s should verify as the default password", h)
		}
	}
	if HasDefaultAdminPassword(hashOf(t, "Other2024!")) {
		t.Error("other password must not match")
	}
}

// Migration 033 must target exactly the default hashes shipped by 029/030 and must not
// stamp password_changed_at (ISS-150).
func TestMigration033TargetsDefaultHashes(t *testing.T) {
	read := func(name string) string {
		b, err := os.ReadFile("../../migrations/" + name)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	up := read("033_force_admin_password_change.up.sql")
	for _, src := range []string{"029_seed_admin.up.sql", "030_reset_admin_password.up.sql"} {
		m := regexp.MustCompile(`'(\$2[ab]\$12\$[^']+)'`).FindStringSubmatch(read(src))
		if m == nil || !strings.Contains(up, m[1]) {
			t.Errorf("033 up does not reference the hash from %s", src)
		}
	}
	if !strings.Contains(up, "force_password_change = true") || strings.Contains(stripSQLComments(up), "password_changed_at") {
		t.Error("033 up must set force_password_change = true and not stamp password_changed_at")
	}
	if !strings.Contains(up, "password_hash IN") {
		t.Error("033 up must be guarded by the default hashes")
	}
}

func stripSQLComments(s string) string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(l), "--") {
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n")
}
