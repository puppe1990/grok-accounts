package authfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

var (
	ErrInvalidJSON = errors.New("auth.json inválido")
	ErrNoIdentity  = errors.New("nenhuma identidade encontrada no auth.json")
)

type Identity struct {
	Key       string
	Email     string
	FirstName string
	LastName  string
	UserID    string
	TeamID    string
	AuthMode  string
	Issuer    string
	ExpiresAt time.Time
}

func (i Identity) DisplayName() string {
	return strings.TrimSpace(i.FirstName + " " + i.LastName)
}

type entry struct {
	AuthMode   string `json:"auth_mode"`
	UserID     string `json:"user_id"`
	Email      string `json:"email"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	TeamID     string `json:"team_id"`
	ExpiresAt  string `json:"expires_at"`
	OIDCIssuer string `json:"oidc_issuer"`
}

func Parse(raw []byte) ([]Identity, error) {
	var entries map[string]entry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidJSON, err)
	}

	ids := make([]Identity, 0, len(entries))
	for key, e := range entries {
		if e.Email == "" {
			continue
		}
		id := Identity{
			Key:       key,
			Email:     e.Email,
			FirstName: e.FirstName,
			LastName:  e.LastName,
			UserID:    e.UserID,
			TeamID:    e.TeamID,
			AuthMode:  e.AuthMode,
			Issuer:    e.OIDCIssuer,
		}
		if id.Issuer == "" {
			id.Issuer, _, _ = strings.Cut(key, "::")
		}
		if ts, err := time.Parse(time.RFC3339Nano, e.ExpiresAt); err == nil {
			id.ExpiresAt = ts
		}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return nil, ErrNoIdentity
	}
	sort.Slice(ids, func(a, b int) bool { return ids[a].Email < ids[b].Email })
	return ids, nil
}
