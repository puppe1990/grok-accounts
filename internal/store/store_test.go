package store

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const devAuth = `{"https://auth.x.ai::client": {"email": "dev@example.com", "auth_mode": "oidc", "key": "tok-dev"}}`

const opsAuth = `{"https://auth.x.ai::client": {"email": "ops@example.com", "auth_mode": "oidc", "key": "tok-ops"}}`

func newStore(t *testing.T) *Store {
	t.Helper()
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	return &Store{Dir: t.TempDir(), Now: func() time.Time { return now }}
}

func TestSaveStoresProfileWithDerivedAlias(t *testing.T) {
	s := newStore(t)

	p, err := s.Save([]byte(devAuth), "")
	if err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}
	if p.Alias != "dev" {
		t.Errorf("Alias = %q, want %q", p.Alias, "dev")
	}
	if p.Email != "dev@example.com" {
		t.Errorf("Email = %q, want %q", p.Email, "dev@example.com")
	}
	if !p.SavedAt.Equal(time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)) {
		t.Errorf("SavedAt = %v, want the injected clock", p.SavedAt)
	}

	reloaded, err := s.Find("dev")
	if err != nil {
		t.Fatalf("Find() error = %v, want nil", err)
	}
	if string(reloaded.Auth) != devAuth {
		t.Errorf("stored auth = %q, want the raw auth.json bytes", reloaded.Auth)
	}
}

func TestSaveSanitizesAliasIntoSlug(t *testing.T) {
	s := newStore(t)

	p, err := s.Save([]byte(devAuth), "Minha Conta/@!")
	if err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}
	if p.Alias != "minha-conta" {
		t.Errorf("Alias = %q, want %q", p.Alias, "minha-conta")
	}
}

func TestSaveRejectsAliasThatSanitizesToEmpty(t *testing.T) {
	s := newStore(t)

	_, err := s.Save([]byte(devAuth), "///")
	if !errors.Is(err, ErrInvalidAlias) {
		t.Errorf("Save() error = %v, want ErrInvalidAlias", err)
	}
}

func TestSaveKeepsProfileInsideStoreDir(t *testing.T) {
	s := newStore(t)

	p, err := s.Save([]byte(devAuth), "../../evil")
	if err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}

	if _, err := os.Stat(filepath.Join(s.Dir, p.Alias+".json")); err != nil {
		t.Errorf("profile file missing inside store dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(s.Dir), "evil.json")); err == nil {
		t.Error("profile escaped the store dir")
	}
}

func TestSaveRefreshesProfileWithSameAliasAndEmail(t *testing.T) {
	s := newStore(t)

	if _, err := s.Save([]byte(devAuth), "work"); err != nil {
		t.Fatalf("first Save() error = %v", err)
	}
	refreshed := `{"https://auth.x.ai::client": {"email": "dev@example.com", "auth_mode": "oidc", "key": "tok-new"}}`
	if _, err := s.Save([]byte(refreshed), "work"); err != nil {
		t.Fatalf("second Save() error = %v, want nil", err)
	}

	p, err := s.Find("work")
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if string(p.Auth) != refreshed {
		t.Errorf("auth = %q, want refreshed snapshot", p.Auth)
	}
}

func TestSaveRejectsAliasTakenByAnotherEmail(t *testing.T) {
	s := newStore(t)

	if _, err := s.Save([]byte(devAuth), "shared"); err != nil {
		t.Fatalf("first Save() error = %v", err)
	}
	_, err := s.Save([]byte(opsAuth), "shared")
	if !errors.Is(err, ErrAliasTaken) {
		t.Errorf("Save() error = %v, want ErrAliasTaken", err)
	}
}

func TestSaveErrorsWhenAuthHasNoIdentity(t *testing.T) {
	s := newStore(t)

	_, err := s.Save([]byte(`{"https://auth.x.ai::client": {"auth_mode": "machine"}}`), "")
	if err == nil {
		t.Error("Save() error = nil, want an error for auth without identity")
	}
}

func TestSaveWritesFilesWithOwnerOnlyPermissions(t *testing.T) {
	s := &Store{Dir: filepath.Join(t.TempDir(), "accounts"), Now: func() time.Time { return time.Now() }}

	if _, err := s.Save([]byte(devAuth), ""); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	info, err := os.Stat(filepath.Join(s.Dir, "dev.json"))
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("profile permissions = %o, want 600", perm)
	}
	dirInfo, err := os.Stat(s.Dir)
	if err != nil {
		t.Fatalf("Stat(dir) error = %v", err)
	}
	if perm := dirInfo.Mode().Perm(); perm != 0o700 {
		t.Errorf("store dir permissions = %o, want 700", perm)
	}
}

func TestListReturnsProfilesSortedByAlias(t *testing.T) {
	s := newStore(t)

	if _, err := s.Save([]byte(opsAuth), "zulu"); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if _, err := s.Save([]byte(devAuth), "alpha"); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	profiles, err := s.List()
	if err != nil {
		t.Fatalf("List() error = %v, want nil", err)
	}
	if len(profiles) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(profiles))
	}
	if profiles[0].Alias != "alpha" || profiles[1].Alias != "zulu" {
		t.Errorf("profiles not sorted: got %q, %q", profiles[0].Alias, profiles[1].Alias)
	}
}

func TestListReturnsEmptyWhenStoreDirIsMissing(t *testing.T) {
	s := &Store{Dir: filepath.Join(t.TempDir(), "does-not-exist")}

	profiles, err := s.List()
	if err != nil {
		t.Fatalf("List() error = %v, want nil", err)
	}
	if len(profiles) != 0 {
		t.Errorf("len(List()) = %d, want 0", len(profiles))
	}
}

func TestListIgnoresNonJSONFiles(t *testing.T) {
	s := newStore(t)
	if _, err := s.Save([]byte(devAuth), ""); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(s.Dir, ".DS_Store"), []byte("junk"), 0o600); err != nil {
		t.Fatalf("setup error = %v", err)
	}

	profiles, err := s.List()
	if err != nil {
		t.Fatalf("List() error = %v, want nil", err)
	}
	if len(profiles) != 1 {
		t.Errorf("len(List()) = %d, want 1", len(profiles))
	}
}

func TestListErrorsOnCorruptProfile(t *testing.T) {
	s := newStore(t)
	if err := os.WriteFile(filepath.Join(s.Dir, "broken.json"), []byte(`{not json`), 0o600); err != nil {
		t.Fatalf("setup error = %v", err)
	}

	if _, err := s.List(); err == nil {
		t.Error("List() error = nil, want an error naming the corrupt profile")
	}
}

func TestFindMatchesAliasIgnoringCase(t *testing.T) {
	s := newStore(t)
	if _, err := s.Save([]byte(devAuth), "Work"); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	p, err := s.Find("WORK")
	if err != nil {
		t.Fatalf("Find() error = %v, want nil", err)
	}
	if p.Alias != "work" {
		t.Errorf("Alias = %q, want %q", p.Alias, "work")
	}
}

func TestFindMatchesByEmailWhenAliasDiffers(t *testing.T) {
	s := newStore(t)
	if _, err := s.Save([]byte(devAuth), "pessoal"); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	p, err := s.Find("dev@example.com")
	if err != nil {
		t.Fatalf("Find() error = %v, want nil", err)
	}
	if p.Alias != "pessoal" {
		t.Errorf("Alias = %q, want %q", p.Alias, "pessoal")
	}
}

func TestFindErrorsWhenNothingMatches(t *testing.T) {
	s := newStore(t)

	_, err := s.Find("ghost")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Find() error = %v, want ErrNotFound", err)
	}
}

func TestRemoveDeletesProfile(t *testing.T) {
	s := newStore(t)
	if _, err := s.Save([]byte(devAuth), ""); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if err := s.Remove("dev"); err != nil {
		t.Fatalf("Remove() error = %v, want nil", err)
	}
	if _, err := s.Find("dev"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Find() after Remove() error = %v, want ErrNotFound", err)
	}
}

func TestRemoveErrorsWhenProfileIsMissing(t *testing.T) {
	s := newStore(t)

	err := s.Remove("ghost")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Remove() error = %v, want ErrNotFound", err)
	}
}
