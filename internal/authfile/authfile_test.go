package authfile

import (
	"errors"
	"testing"
	"time"
)

const sampleAuth = `{
  "https://auth.x.ai::b1a00492-073a-47ea-816f-4c329264a828": {
    "key": "session-token-value",
    "auth_mode": "oidc",
    "create_time": "2026-09-16T13:41:13.785343Z",
    "user_id": "a64487c2-a90a-443d-b2ff-56eca73e81e6",
    "email": "dev@example.com",
    "first_name": "Dev",
    "last_name": "Example",
    "principal_type": "User",
    "principal_id": "a64487c2-a90a-443d-b2ff-56eca73e81e6",
    "team_id": "6206d352-6cd2-4370-b42f-5d5c9fe87b91",
    "refresh_token": "refresh-token-value",
    "expires_at": "2026-09-16T19:41:13.785343Z",
    "oidc_issuer": "https://auth.x.ai",
    "oidc_client_id": "b1a00492-073a-47ea-816f-4c329264a828"
  }
}`

func TestParseExtractsIdentity(t *testing.T) {
	ids, err := Parse([]byte(sampleAuth))
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if len(ids) != 1 {
		t.Fatalf("len(Parse()) = %d, want 1", len(ids))
	}

	id := ids[0]
	if id.Email != "dev@example.com" {
		t.Errorf("Email = %q, want %q", id.Email, "dev@example.com")
	}
	if got, want := id.DisplayName(), "Dev Example"; got != want {
		t.Errorf("DisplayName() = %q, want %q", got, want)
	}
	if id.TeamID != "6206d352-6cd2-4370-b42f-5d5c9fe87b91" {
		t.Errorf("TeamID = %q, want %q", id.TeamID, "6206d352-6cd2-4370-b42f-5d5c9fe87b91")
	}
	if id.AuthMode != "oidc" {
		t.Errorf("AuthMode = %q, want %q", id.AuthMode, "oidc")
	}
	if id.Issuer != "https://auth.x.ai" {
		t.Errorf("Issuer = %q, want %q", id.Issuer, "https://auth.x.ai")
	}
	if id.Key != "https://auth.x.ai::b1a00492-073a-47ea-816f-4c329264a828" {
		t.Errorf("Key = %q, want the auth.json map key", id.Key)
	}
	want := time.Date(2026, 9, 16, 19, 41, 13, 785343000, time.UTC)
	if !id.ExpiresAt.Equal(want) {
		t.Errorf("ExpiresAt = %v, want %v", id.ExpiresAt, want)
	}
}

func TestParseSortsIdentitiesByEmail(t *testing.T) {
	raw := `{
      "https://auth.x.ai::client": {"email": "zoe@example.com", "auth_mode": "oidc"},
      "https://other.idp::client": {"email": "ana@example.com", "auth_mode": "oidc"}
    }`

	ids, err := Parse([]byte(raw))
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if len(ids) != 2 {
		t.Fatalf("len(Parse()) = %d, want 2", len(ids))
	}
	if ids[0].Email != "ana@example.com" || ids[1].Email != "zoe@example.com" {
		t.Errorf("identities not sorted by email: got %q, %q", ids[0].Email, ids[1].Email)
	}
}

func TestParseFallsBackToKeyPrefixForIssuer(t *testing.T) {
	raw := `{"https://auth.x.ai::client": {"email": "dev@example.com"}}`

	ids, err := Parse([]byte(raw))
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if ids[0].Issuer != "https://auth.x.ai" {
		t.Errorf("Issuer = %q, want %q", ids[0].Issuer, "https://auth.x.ai")
	}
}

func TestParseSkipsEntriesWithoutEmail(t *testing.T) {
	raw := `{
      "https://auth.x.ai::client": {"email": "dev@example.com"},
      "https://auth.x.ai::machine": {"principal_type": "Machine", "auth_mode": "machine"}
    }`

	ids, err := Parse([]byte(raw))
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if len(ids) != 1 {
		t.Fatalf("len(Parse()) = %d, want 1", len(ids))
	}
	if ids[0].Email != "dev@example.com" {
		t.Errorf("Email = %q, want %q", ids[0].Email, "dev@example.com")
	}
}

func TestParseIgnoresUnparsableExpiry(t *testing.T) {
	raw := `{"https://auth.x.ai::client": {"email": "dev@example.com", "expires_at": "not-a-timestamp"}}`

	ids, err := Parse([]byte(raw))
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if !ids[0].ExpiresAt.IsZero() {
		t.Errorf("ExpiresAt = %v, want zero time", ids[0].ExpiresAt)
	}
}

func TestParseErrorsOnInvalidJSON(t *testing.T) {
	_, err := Parse([]byte(`{"broken":`))
	if !errors.Is(err, ErrInvalidJSON) {
		t.Errorf("Parse() error = %v, want ErrInvalidJSON", err)
	}
}

func TestParseErrorsWhenNoIdentityPresent(t *testing.T) {
	_, err := Parse([]byte(`{}`))
	if !errors.Is(err, ErrNoIdentity) {
		t.Errorf("Parse() error = %v, want ErrNoIdentity", err)
	}
}

func TestDisplayNameDropsTrailingSurnameRepeatedFromFirstName(t *testing.T) {
	id := Identity{FirstName: "Gabriel F Dos Santos", LastName: "Fontoura Dos Santos"}

	if got, want := id.DisplayName(), "Gabriel F Dos Santos Fontoura"; got != want {
		t.Errorf("DisplayName() = %q, want %q", got, want)
	}
}

func TestDisplayNameDropsSurnameEqualToOneInFirstName(t *testing.T) {
	id := Identity{FirstName: "Maria Silva", LastName: "Silva"}

	if got, want := id.DisplayName(), "Maria Silva"; got != want {
		t.Errorf("DisplayName() = %q, want %q", got, want)
	}
}

func TestDisplayNameDeduplicatesIgnoringCase(t *testing.T) {
	id := Identity{FirstName: "maria silva", LastName: "Silva"}

	if got, want := id.DisplayName(), "maria silva"; got != want {
		t.Errorf("DisplayName() = %q, want %q", got, want)
	}
}

func TestDisplayNameMergesOverlappingTokens(t *testing.T) {
	id := Identity{FirstName: "Joao Pedro", LastName: "Pedro Almeida"}

	if got, want := id.DisplayName(), "Joao Pedro Almeida"; got != want {
		t.Errorf("DisplayName() = %q, want %q", got, want)
	}
}

func TestDisplayNameHandlesRepeatedNameInBothFields(t *testing.T) {
	id := Identity{FirstName: "Dev", LastName: "Dev"}

	if got, want := id.DisplayName(), "Dev"; got != want {
		t.Errorf("DisplayName() = %q, want %q", got, want)
	}
}

func TestDisplayNameUsesWhicheverFieldIsPresent(t *testing.T) {
	if got, want := (Identity{FirstName: "Dev"}).DisplayName(), "Dev"; got != want {
		t.Errorf("DisplayName() = %q, want %q", got, want)
	}
	if got, want := (Identity{LastName: "Example"}).DisplayName(), "Example"; got != want {
		t.Errorf("DisplayName() = %q, want %q", got, want)
	}
}
