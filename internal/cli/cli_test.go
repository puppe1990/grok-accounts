package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/puppe1990/grok-accounts/internal/store"
)

const devAuth = `{
  "https://auth.x.ai::client": {
    "email": "dev@example.com",
    "first_name": "Dev",
    "last_name": "Example",
    "auth_mode": "oidc",
    "team_id": "team-dev",
    "key": "tok-dev",
    "expires_at": "2026-09-16T19:41:13Z"
  }
}`

const opsAuth = `{"https://auth.x.ai::client": {"email": "ops@example.com", "auth_mode": "oidc", "key": "tok-ops"}}`

func newEnv(t *testing.T) (Env, *bytes.Buffer, *bytes.Buffer, string) {
	t.Helper()
	home := t.TempDir()
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	env := Env{
		GrokHome: home,
		Stdout:   out,
		Stderr:   errOut,
		Now:      func() time.Time { return time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC) },
	}
	return env, out, errOut, home
}

func writeLive(t *testing.T, home, raw string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(home, "auth.json"), []byte(raw), 0o600); err != nil {
		t.Fatalf("setup: write auth.json: %v", err)
	}
}

func newStore(home string) *store.Store {
	return &store.Store{Dir: filepath.Join(home, "accounts")}
}

func saveProfile(t *testing.T, home, raw, alias string) {
	t.Helper()
	if _, err := newStore(home).Save([]byte(raw), alias); err != nil {
		t.Fatalf("setup: save profile %q: %v", alias, err)
	}
}

func TestRunListShowsProfilesAndMarksActiveAccount(t *testing.T) {
	env, out, _, home := newEnv(t)
	saveProfile(t, home, opsAuth, "ops")
	saveProfile(t, home, devAuth, "dev")
	writeLive(t, home, devAuth)

	code := Run([]string{"list"}, env)

	if code != 0 {
		t.Fatalf("Run(list) = %d, want 0 (stderr: %s)", code, env.Stderr)
	}
	got := out.String()
	for _, want := range []string{"ALIAS", "dev", "ops", "dev@example.com", "ops@example.com"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
	if !strings.Contains(got, "* dev") {
		t.Errorf("active profile not marked with *:\n%s", got)
	}
	if strings.Contains(got, "* ops") {
		t.Errorf("inactive profile marked as active:\n%s", got)
	}
}

func TestRunListOnEmptyStorePointsToAdd(t *testing.T) {
	env, out, _, _ := newEnv(t)

	code := Run([]string{"list"}, env)

	if code != 0 {
		t.Fatalf("Run(list) = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "grok-accounts add") {
		t.Errorf("output should hint at the add command:\n%s", out.String())
	}
}

func TestRunCurrentShowsLiveAccount(t *testing.T) {
	env, out, _, home := newEnv(t)
	writeLive(t, home, devAuth)

	code := Run([]string{"current"}, env)

	if code != 0 {
		t.Fatalf("Run(current) = %d, want 0 (stderr: %s)", code, env.Stderr)
	}
	got := out.String()
	if !strings.Contains(got, "dev@example.com") {
		t.Errorf("output missing email:\n%s", got)
	}
	if !strings.Contains(got, "2026-09-16 19:41") {
		t.Errorf("output missing expiry:\n%s", got)
	}
}

func TestRunCurrentFailsWhenNoAccountIsLoggedIn(t *testing.T) {
	env, _, errOut, _ := newEnv(t)

	code := Run([]string{"current"}, env)

	if code != 1 {
		t.Fatalf("Run(current) = %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "Nenhuma conta logada") {
		t.Errorf("stderr = %q, want a not-logged-in message", errOut.String())
	}
}

func TestRunCurrentFailsOnCorruptAuthFile(t *testing.T) {
	env, _, errOut, home := newEnv(t)
	writeLive(t, home, "{broken")

	code := Run([]string{"current"}, env)

	if code != 1 {
		t.Fatalf("Run(current) = %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "auth.json") {
		t.Errorf("stderr = %q, want an auth.json error", errOut.String())
	}
}

func TestRunAddSnapshotsLiveAccount(t *testing.T) {
	env, out, _, home := newEnv(t)
	writeLive(t, home, devAuth)

	code := Run([]string{"add"}, env)

	if code != 0 {
		t.Fatalf("Run(add) = %d, want 0 (stderr: %s)", code, env.Stderr)
	}
	if !strings.Contains(out.String(), "dev") {
		t.Errorf("output = %q, want the saved alias", out.String())
	}

	p, err := newStore(home).Find("dev")
	if err != nil {
		t.Fatalf("Find(dev) error = %v, want the snapshot to exist", err)
	}
	if string(p.Auth) != devAuth {
		t.Errorf("stored auth = %q, want live auth.json bytes", p.Auth)
	}
}

func TestRunAddUsesAliasFlag(t *testing.T) {
	env, _, _, home := newEnv(t)
	writeLive(t, home, devAuth)

	code := Run([]string{"add", "--alias", "trabalho"}, env)

	if code != 0 {
		t.Fatalf("Run(add --alias) = %d, want 0 (stderr: %s)", code, env.Stderr)
	}
	if _, err := newStore(home).Find("trabalho"); err != nil {
		t.Errorf("Find(trabalho) error = %v, want profile with custom alias", err)
	}
}

func TestRunAddFailsWhenAliasBelongsToAnotherAccount(t *testing.T) {
	env, _, errOut, home := newEnv(t)
	saveProfile(t, home, opsAuth, "trabalho")
	writeLive(t, home, devAuth)

	code := Run([]string{"add", "--alias", "trabalho"}, env)

	if code != 1 {
		t.Fatalf("Run(add) = %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "apelido") {
		t.Errorf("stderr = %q, want an alias-taken message", errOut.String())
	}
}

func TestRunAddFailsWhenNotLoggedIn(t *testing.T) {
	env, _, errOut, _ := newEnv(t)

	code := Run([]string{"add"}, env)

	if code != 1 {
		t.Fatalf("Run(add) = %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "Nenhuma conta logada") {
		t.Errorf("stderr = %q, want a not-logged-in message", errOut.String())
	}
}

func TestRunSwitchReplacesLiveAuthFile(t *testing.T) {
	env, out, _, home := newEnv(t)
	saveProfile(t, home, opsAuth, "ops")
	saveProfile(t, home, devAuth, "dev")
	writeLive(t, home, devAuth)

	code := Run([]string{"switch", "ops"}, env)

	if code != 0 {
		t.Fatalf("Run(switch) = %d, want 0 (stderr: %s)", code, env.Stderr)
	}
	if !strings.Contains(out.String(), "ops@example.com") {
		t.Errorf("output = %q, want the new active account", out.String())
	}

	live, err := os.ReadFile(filepath.Join(home, "auth.json"))
	if err != nil {
		t.Fatalf("read auth.json: %v", err)
	}
	if string(live) != opsAuth {
		t.Errorf("auth.json = %q, want %q", live, opsAuth)
	}
	info, err := os.Stat(filepath.Join(home, "auth.json"))
	if err != nil {
		t.Fatalf("stat auth.json: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("auth.json permissions = %o, want 600", perm)
	}
}

func TestRunSwitchRestoresProfileContent(t *testing.T) {
	env, _, _, home := newEnv(t)
	raw := devAuth + "\n"
	saveProfile(t, home, raw, "dev")
	writeLive(t, home, opsAuth)

	code := Run([]string{"switch", "dev"}, env)

	if code != 0 {
		t.Fatalf("Run(switch) = %d, want 0 (stderr: %s)", code, env.Stderr)
	}
	live, err := os.ReadFile(filepath.Join(home, "auth.json"))
	if err != nil {
		t.Fatalf("read auth.json: %v", err)
	}
	if got, want := strings.TrimSpace(string(live)), strings.TrimSpace(raw); got != want {
		t.Errorf("auth.json = %q, want %q", got, want)
	}
	for _, want := range []string{"tok-dev", "team-dev", "2026-09-16T19:41:13Z"} {
		if !strings.Contains(string(live), want) {
			t.Errorf("auth.json lost %q after the switch:\n%s", want, live)
		}
	}
}

func TestRunSwitchSavesPreviousAccountFirst(t *testing.T) {
	env, out, _, home := newEnv(t)
	saveProfile(t, home, opsAuth, "ops")
	writeLive(t, home, devAuth)

	code := Run([]string{"switch", "ops"}, env)

	if code != 0 {
		t.Fatalf("Run(switch) = %d, want 0 (stderr: %s)", code, env.Stderr)
	}
	if !strings.Contains(out.String(), "dev") {
		t.Errorf("output = %q, want a note about the snapshot of the previous account", out.String())
	}
	if _, err := newStore(home).Find("dev@example.com"); err != nil {
		t.Errorf("previous account was not snapshotted: %v", err)
	}
}

func TestRunSwitchRefreshesExistingProfileOfPreviousAccount(t *testing.T) {
	env, _, _, home := newEnv(t)
	saveProfile(t, home, opsAuth, "ops")
	saveProfile(t, home, devAuth, "pessoal")
	writeLive(t, home, devAuth)

	if code := Run([]string{"switch", "ops"}, env); code != 0 {
		t.Fatalf("Run(switch) = %d, want 0 (stderr: %s)", code, env.Stderr)
	}

	profiles, err := newStore(home).List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(profiles) != 2 {
		t.Fatalf("len(List()) = %d, want 2 (no duplicate for the same account)", len(profiles))
	}
}

func TestRunSwitchToAlreadyActiveAccountChangesNothing(t *testing.T) {
	env, out, _, home := newEnv(t)
	saveProfile(t, home, devAuth, "dev")
	writeLive(t, home, devAuth)

	code := Run([]string{"switch", "dev"}, env)

	if code != 0 {
		t.Fatalf("Run(switch) = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "já é a conta ativa") {
		t.Errorf("output = %q, want an already-active message", out.String())
	}
	live, err := os.ReadFile(filepath.Join(home, "auth.json"))
	if err != nil {
		t.Fatalf("read auth.json: %v", err)
	}
	if string(live) != devAuth {
		t.Errorf("auth.json changed to %q", live)
	}
}

func TestRunSwitchFailsWhenProfileIsMissing(t *testing.T) {
	env, _, errOut, home := newEnv(t)
	writeLive(t, home, devAuth)

	code := Run([]string{"switch", "ghost"}, env)

	if code != 1 {
		t.Fatalf("Run(switch) = %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "ghost") {
		t.Errorf("stderr = %q, want the missing reference", errOut.String())
	}
}

func TestRunSwitchRequiresArgument(t *testing.T) {
	env, _, errOut, _ := newEnv(t)

	code := Run([]string{"switch"}, env)

	if code != 2 {
		t.Fatalf("Run(switch) = %d, want 2", code)
	}
	if !strings.Contains(errOut.String(), "uso:") {
		t.Errorf("stderr = %q, want usage", errOut.String())
	}
}

func TestRunSwitchReplacesCorruptAuthFileWithWarning(t *testing.T) {
	env, _, errOut, home := newEnv(t)
	saveProfile(t, home, opsAuth, "ops")
	writeLive(t, home, "{broken")

	code := Run([]string{"switch", "ops"}, env)

	if code != 0 {
		t.Fatalf("Run(switch) = %d, want 0 (stderr: %s)", code, env.Stderr)
	}
	if !strings.Contains(errOut.String(), "aviso") {
		t.Errorf("stderr = %q, want a warning about the unreadable auth.json", errOut.String())
	}
	live, err := os.ReadFile(filepath.Join(home, "auth.json"))
	if err != nil {
		t.Fatalf("read auth.json: %v", err)
	}
	if string(live) != opsAuth {
		t.Errorf("auth.json = %q, want the profile to replace the corrupt file", live)
	}
}

func TestRunRemoveDeletesProfile(t *testing.T) {
	env, out, _, home := newEnv(t)
	saveProfile(t, home, opsAuth, "ops")

	code := Run([]string{"remove", "ops"}, env)

	if code != 0 {
		t.Fatalf("Run(remove) = %d, want 0 (stderr: %s)", code, env.Stderr)
	}
	if !strings.Contains(out.String(), "ops") {
		t.Errorf("output = %q, want the removed alias", out.String())
	}
	if _, err := newStore(home).Find("ops"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("Find(ops) error = %v, want the profile to be gone", err)
	}
}

func TestRunRemoveFailsWhenProfileIsMissing(t *testing.T) {
	env, _, errOut, _ := newEnv(t)

	code := Run([]string{"remove", "ghost"}, env)

	if code != 1 {
		t.Fatalf("Run(remove) = %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "ghost") {
		t.Errorf("stderr = %q, want the missing reference", errOut.String())
	}
}

func TestRunLoginSnapshotsPreviousAndNewAccounts(t *testing.T) {
	env, out, _, home := newEnv(t)
	writeLive(t, home, devAuth)
	env.RunLogin = func(deviceAuth bool) error {
		return os.WriteFile(filepath.Join(home, "auth.json"), []byte(opsAuth), 0o600)
	}

	code := Run([]string{"login", "--alias", "cliente"}, env)

	if code != 0 {
		t.Fatalf("Run(login) = %d, want 0 (stderr: %s)", code, env.Stderr)
	}
	if !strings.Contains(out.String(), "ops@example.com") {
		t.Errorf("output = %q, want the freshly logged-in account", out.String())
	}

	s := newStore(home)
	if _, err := s.Find("dev"); err != nil {
		t.Errorf("previous account was not snapshotted: %v", err)
	}
	if p, err := s.Find("cliente"); err != nil {
		t.Errorf("new account was not snapshotted: %v", err)
	} else if p.Email != "ops@example.com" {
		t.Errorf("profile cliente = %q, want ops@example.com", p.Email)
	}
}

func TestRunLoginUsesDerivedAliasWithoutFlag(t *testing.T) {
	env, _, _, home := newEnv(t)
	env.RunLogin = func(deviceAuth bool) error {
		return os.WriteFile(filepath.Join(home, "auth.json"), []byte(opsAuth), 0o600)
	}

	if code := Run([]string{"login"}, env); code != 0 {
		t.Fatalf("Run(login) = %d, want 0 (stderr: %s)", code, env.Stderr)
	}
	if _, err := newStore(home).Find("ops"); err != nil {
		t.Errorf("Find(ops) error = %v, want profile derived from the email", err)
	}
}

func TestRunLoginPassesDeviceAuthFlag(t *testing.T) {
	env, _, _, home := newEnv(t)
	var gotDeviceAuth bool
	env.RunLogin = func(deviceAuth bool) error {
		gotDeviceAuth = deviceAuth
		return os.WriteFile(filepath.Join(home, "auth.json"), []byte(opsAuth), 0o600)
	}

	if code := Run([]string{"login", "--device-auth"}, env); code != 0 {
		t.Fatalf("Run(login --device-auth) = %d, want 0 (stderr: %s)", code, env.Stderr)
	}
	if !gotDeviceAuth {
		t.Error("RunLogin() was not called with deviceAuth=true")
	}
}

func TestRunLoginFailsWhenGrokLoginFails(t *testing.T) {
	env, _, errOut, home := newEnv(t)
	writeLive(t, home, devAuth)
	env.RunLogin = func(deviceAuth bool) error { return errors.New("exit status 1") }

	code := Run([]string{"login"}, env)

	if code != 1 {
		t.Fatalf("Run(login) = %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "grok login") {
		t.Errorf("stderr = %q, want a login failure message", errOut.String())
	}
	if _, err := newStore(home).Find("dev"); err != nil {
		t.Errorf("account active before login was not snapshotted: %v", err)
	}
}

func TestRunUnknownCommandPrintsUsageToStderr(t *testing.T) {
	env, out, errOut, _ := newEnv(t)

	code := Run([]string{"wat"}, env)

	if code != 2 {
		t.Fatalf("Run(wat) = %d, want 2", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout = %q, want empty", out.String())
	}
	if !strings.Contains(errOut.String(), "uso:") {
		t.Errorf("stderr = %q, want usage", errOut.String())
	}
}

func TestRunWithoutArgumentsPrintsUsage(t *testing.T) {
	env, out, _, _ := newEnv(t)

	code := Run(nil, env)

	if code != 0 {
		t.Fatalf("Run(nil) = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "uso:") {
		t.Errorf("stdout = %q, want usage", out.String())
	}
}

func TestResolveGrokHomePrefersEnvVariable(t *testing.T) {
	got, err := ResolveGrokHome(func(key string) string {
		if key == "GROK_HOME" {
			return "/custom/grok"
		}
		return "/home/user"
	})
	if err != nil {
		t.Fatalf("ResolveGrokHome() error = %v, want nil", err)
	}
	if got != "/custom/grok" {
		t.Errorf("ResolveGrokHome() = %q, want %q", got, "/custom/grok")
	}
}

func TestResolveGrokHomeFallsBackToHomeDir(t *testing.T) {
	got, err := ResolveGrokHome(func(key string) string {
		if key == "HOME" {
			return "/home/user"
		}
		return ""
	})
	if err != nil {
		t.Fatalf("ResolveGrokHome() error = %v, want nil", err)
	}
	if want := filepath.Join("/home/user", ".grok"); got != want {
		t.Errorf("ResolveGrokHome() = %q, want %q", got, want)
	}
}

func TestResolveGrokHomeFailsWithoutHome(t *testing.T) {
	_, err := ResolveGrokHome(func(string) string { return "" })
	if err == nil {
		t.Error("ResolveGrokHome() error = nil, want an error")
	}
}
