package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/puppe1990/grok-accounts/internal/authfile"
)

var (
	ErrInvalidAlias = errors.New("apelido inválido")
	ErrAliasTaken   = errors.New("apelido já usado por outra conta")
	ErrNotFound     = errors.New("perfil não encontrado")
)

type Profile struct {
	Alias   string          `json:"alias"`
	Email   string          `json:"email"`
	SavedAt time.Time       `json:"saved_at"`
	Auth    json.RawMessage `json:"auth"`
}

type Store struct {
	Dir string
	Now func() time.Time
}

func (s *Store) Save(raw []byte, alias string) (Profile, error) {
	ids, err := authfile.Parse(raw)
	if err != nil {
		return Profile{}, err
	}
	identity := ids[0]

	if alias == "" {
		alias = Slug(identity.Email[:strings.Index(identity.Email+"@", "@")])
	}
	alias = Slug(alias)
	if alias == "" {
		return Profile{}, ErrInvalidAlias
	}

	existing, err := s.Find(alias)
	switch {
	case err == nil:
		if !strings.EqualFold(existing.Email, identity.Email) {
			return Profile{}, fmt.Errorf("%w: %q", ErrAliasTaken, alias)
		}
	case !errors.Is(err, ErrNotFound):
		return Profile{}, err
	}

	profile := Profile{
		Alias:   alias,
		Email:   identity.Email,
		SavedAt: s.now(),
		Auth:    bytes.TrimSpace(append(json.RawMessage(nil), raw...)),
	}
	data, err := encode(profile)
	if err != nil {
		return Profile{}, err
	}
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return Profile{}, err
	}
	if err := authfile.WriteAtomic(s.path(alias), data); err != nil {
		return Profile{}, err
	}
	return profile, nil
}

func (s *Store) List() ([]Profile, error) {
	entries, err := os.ReadDir(s.Dir)
	if errors.Is(err, fs.ErrNotExist) {
		return []Profile{}, nil
	}
	if err != nil {
		return nil, err
	}

	profiles := make([]Profile, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || strings.HasPrefix(name, ".") || !strings.HasSuffix(name, ".json") {
			continue
		}
		p, err := s.load(filepath.Join(s.Dir, name))
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	sort.Slice(profiles, func(i, j int) bool { return profiles[i].Alias < profiles[j].Alias })
	return profiles, nil
}

func (s *Store) Find(ref string) (Profile, error) {
	profiles, err := s.List()
	if err != nil {
		return Profile{}, err
	}
	slug := Slug(ref)
	for _, p := range profiles {
		if p.Alias == slug || strings.EqualFold(p.Email, strings.TrimSpace(ref)) {
			return p, nil
		}
	}
	return Profile{}, fmt.Errorf("%w: %q", ErrNotFound, ref)
}

func (s *Store) Remove(ref string) error {
	p, err := s.Find(ref)
	if err != nil {
		return err
	}
	return os.Remove(s.path(p.Alias))
}

func encode(p Profile) ([]byte, error) {
	alias, err := json.Marshal(p.Alias)
	if err != nil {
		return nil, err
	}
	email, err := json.Marshal(p.Email)
	if err != nil {
		return nil, err
	}
	savedAt, err := json.Marshal(p.SavedAt)
	if err != nil {
		return nil, err
	}
	auth := bytes.TrimSpace(p.Auth)
	if len(auth) == 0 {
		auth = []byte("null")
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "{\n  \"alias\": %s,\n  \"email\": %s,\n  \"saved_at\": %s,\n  \"auth\": %s\n}\n", alias, email, savedAt, auth)
	return buf.Bytes(), nil
}

func (s *Store) load(path string) (Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Profile{}, err
	}
	var p Profile
	if err := json.Unmarshal(data, &p); err != nil {
		return Profile{}, fmt.Errorf("perfil corrompido %s: %w", filepath.Base(path), err)
	}
	return p, nil
}

func (s *Store) path(alias string) string {
	return filepath.Join(s.Dir, alias+".json")
}

func (s *Store) now() time.Time {
	if s.Now == nil {
		return time.Now()
	}
	return s.Now()
}

func Slug(s string) string {
	var b strings.Builder
	pendingDash := false
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			if pendingDash {
				b.WriteByte('-')
				pendingDash = false
			}
			b.WriteRune(r)
			continue
		}
		if b.Len() > 0 {
			pendingDash = true
		}
	}
	return b.String()
}
