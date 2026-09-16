package cli

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/puppe1990/grok-accounts/internal/authfile"
	"github.com/puppe1990/grok-accounts/internal/store"
)

type liveState struct {
	Raw      []byte
	ID       authfile.Identity
	HasID    bool
	ReadErr  error
	ParseErr error
}

func (env Env) live() liveState {
	raw, err := authfile.Read(env.authPath())
	if errors.Is(err, fs.ErrNotExist) {
		return liveState{}
	}
	if err != nil {
		return liveState{ReadErr: err}
	}
	ids, err := authfile.Parse(raw)
	if err != nil {
		return liveState{Raw: raw, ParseErr: err}
	}
	return liveState{Raw: raw, ID: ids[0], HasID: true}
}

func (env Env) newFlags(name string) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(env.Stderr)
	return flags
}

func runList(args []string, env Env) int {
	flags := env.newFlags("list")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(env.Stderr, "uso: grok-accounts list\n")
		return 2
	}

	profiles, err := env.store().List()
	if err != nil {
		fmt.Fprintf(env.Stderr, "erro ao ler os perfis: %v\n", err)
		return 1
	}
	if len(profiles) == 0 {
		fmt.Fprintln(env.Stdout, "Nenhum perfil salvo. Use: grok-accounts add")
		return 0
	}

	live := env.live()
	tw := tabwriter.NewWriter(env.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "ALIAS\tCONTA\tEXPIRA")
	for _, p := range profiles {
		label := "  " + p.Alias
		if live.HasID && strings.EqualFold(live.ID.Email, p.Email) {
			label = "* " + p.Alias
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n", label, p.Email, profileExpiry(p))
	}
	tw.Flush()
	return 0
}

func runCurrent(args []string, env Env) int {
	flags := env.newFlags("current")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(env.Stderr, "uso: grok-accounts current\n")
		return 2
	}

	live := env.live()
	if live.ReadErr != nil {
		fmt.Fprintf(env.Stderr, "erro ao ler o auth.json: %v\n", live.ReadErr)
		return 1
	}
	if !live.HasID {
		if live.ParseErr != nil {
			fmt.Fprintf(env.Stderr, "auth.json ilegível: %v\n", live.ParseErr)
			return 1
		}
		fmt.Fprintln(env.Stderr, "Nenhuma conta logada. Faça login com: grok login")
		return 1
	}

	fmt.Fprintf(env.Stdout, "Conta ativa: %s\n", live.ID.Email)
	if name := live.ID.DisplayName(); name != "" {
		fmt.Fprintf(env.Stdout, "Nome: %s\n", name)
	}
	fmt.Fprintf(env.Stdout, "Sessão: %s\n", describe(live.ID))
	return 0
}

func runAdd(args []string, env Env) int {
	flags := env.newFlags("add")
	alias := flags.String("alias", "", "apelido do perfil")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(env.Stderr, "uso: grok-accounts add [--alias NOME]\n")
		return 2
	}

	live := env.live()
	if code, ok := requireLive(live, env); !ok {
		return code
	}

	profile, err := env.store().Save(live.Raw, *alias)
	if err != nil {
		return reportStoreError(err, env)
	}
	fmt.Fprintf(env.Stdout, "Perfil salvo: %s (%s)\n", profile.Alias, profile.Email)
	return 0
}

func runLogin(args []string, env Env) int {
	flags := env.newFlags("login")
	alias := flags.String("alias", "", "apelido do perfil")
	deviceAuth := flags.Bool("device-auth", false, "login por código de dispositivo (SSH/remoto)")
	deviceCode := flags.Bool("device-code", false, "alias de --device-auth")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(env.Stderr, "uso: grok-accounts login [--alias NOME] [--device-auth]\n")
		return 2
	}
	if env.RunLogin == nil {
		fmt.Fprintln(env.Stderr, "login indisponível: nenhum executor configurado")
		return 1
	}

	live := env.live()
	if live.ReadErr != nil {
		fmt.Fprintf(env.Stderr, "erro ao ler o auth.json: %v\n", live.ReadErr)
		return 1
	}
	if live.HasID {
		previous, err := autoSave(env.store(), live.Raw)
		if err != nil {
			fmt.Fprintf(env.Stderr, "aviso: não foi possível salvar a conta atual (%v)\n", err)
		} else {
			fmt.Fprintf(env.Stdout, "Conta atual guardada no perfil %q.\n", previous)
		}
	}

	if err := env.RunLogin(*deviceAuth || *deviceCode); err != nil {
		fmt.Fprintf(env.Stderr, "grok login falhou: %v\n", err)
		return 1
	}

	live = env.live()
	if code, ok := requireLive(live, env); !ok {
		return code
	}

	profile, err := env.store().Save(live.Raw, *alias)
	if err != nil {
		return reportStoreError(err, env)
	}
	fmt.Fprintf(env.Stdout, "Conta ativa agora: %s (perfil %q)\n", profile.Email, profile.Alias)
	return 0
}

func runSwitch(args []string, env Env) int {
	if len(args) != 1 {
		fmt.Fprintln(env.Stderr, "uso: grok-accounts switch <perfil|email>")
		return 2
	}

	st := env.store()
	profile, err := st.Find(args[0])
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			fmt.Fprintf(env.Stderr, "perfil não encontrado: %q\n", args[0])
			return 1
		}
		fmt.Fprintf(env.Stderr, "erro ao ler os perfis: %v\n", err)
		return 1
	}

	live := env.live()
	if live.ReadErr != nil {
		fmt.Fprintf(env.Stderr, "erro ao ler o auth.json: %v\n", live.ReadErr)
		return 1
	}

	switch {
	case live.HasID && strings.EqualFold(live.ID.Email, profile.Email):
		fmt.Fprintf(env.Stdout, "A conta %s já é a conta ativa (perfil %q).\n", profile.Email, profile.Alias)
		return 0
	case live.HasID:
		previous, err := autoSave(st, live.Raw)
		if err != nil {
			fmt.Fprintf(env.Stderr, "erro ao salvar a conta atual antes de trocar: %v\n", err)
			return 1
		}
		fmt.Fprintf(env.Stdout, "Conta anterior guardada no perfil %q.\n", previous)
	case live.ParseErr != nil:
		fmt.Fprintln(env.Stderr, "aviso: o auth.json atual não pôde ser lido; ele será substituído.")
	}

	if err := authfile.WriteAtomic(env.authPath(), profile.Auth); err != nil {
		fmt.Fprintf(env.Stderr, "erro ao gravar o auth.json: %v\n", err)
		return 1
	}
	fmt.Fprintf(env.Stdout, "Conta ativa agora: %s (perfil %q)\n", profile.Email, profile.Alias)
	fmt.Fprintln(env.Stdout, "O grok recarrega o auth.json sozinho — não precisa reiniciar o CLI.")
	return 0
}

func runRemove(args []string, env Env) int {
	if len(args) != 1 {
		fmt.Fprintln(env.Stderr, "uso: grok-accounts remove <perfil|email>")
		return 2
	}

	st := env.store()
	profile, err := st.Find(args[0])
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			fmt.Fprintf(env.Stderr, "perfil não encontrado: %q\n", args[0])
			return 1
		}
		fmt.Fprintf(env.Stderr, "erro ao ler os perfis: %v\n", err)
		return 1
	}
	if err := st.Remove(profile.Alias); err != nil {
		fmt.Fprintf(env.Stderr, "erro ao remover o perfil: %v\n", err)
		return 1
	}

	fmt.Fprintf(env.Stdout, "Perfil removido: %s (%s)\n", profile.Alias, profile.Email)
	live := env.live()
	if live.HasID && strings.EqualFold(live.ID.Email, profile.Email) {
		fmt.Fprintln(env.Stdout, "A sessão ativa segue válida: o auth.json não foi alterado.")
	}
	return 0
}

func autoSave(st *store.Store, raw []byte) (string, error) {
	ids, err := authfile.Parse(raw)
	if err != nil {
		return "", err
	}

	existing, err := st.Find(ids[0].Email)
	switch {
	case err == nil:
		if _, err := st.Save(raw, existing.Alias); err != nil {
			return "", err
		}
		return existing.Alias, nil
	case !errors.Is(err, store.ErrNotFound):
		return "", err
	}

	profile, err := st.Save(raw, "")
	if errors.Is(err, store.ErrAliasTaken) {
		profile, err = st.Save(raw, store.Slug(ids[0].Email))
	}
	if err != nil {
		return "", err
	}
	return profile.Alias, nil
}

func requireLive(live liveState, env Env) (int, bool) {
	switch {
	case live.ReadErr != nil:
		fmt.Fprintf(env.Stderr, "erro ao ler o auth.json: %v\n", live.ReadErr)
		return 1, false
	case live.HasID:
		return 0, true
	case live.ParseErr != nil:
		fmt.Fprintf(env.Stderr, "auth.json ilegível: %v\n", live.ParseErr)
		return 1, false
	default:
		fmt.Fprintln(env.Stderr, "Nenhuma conta logada. Faça login com: grok login")
		return 1, false
	}
}

func reportStoreError(err error, env Env) int {
	if errors.Is(err, store.ErrAliasTaken) {
		fmt.Fprintf(env.Stderr, "%v — escolha outro com --alias\n", err)
		return 1
	}
	fmt.Fprintf(env.Stderr, "erro ao salvar o perfil: %v\n", err)
	return 1
}

func profileExpiry(p store.Profile) string {
	ids, err := authfile.Parse(p.Auth)
	if err != nil {
		return "-"
	}
	return formatExpiry(ids[0].ExpiresAt)
}

func describe(id authfile.Identity) string {
	parts := make([]string, 0, 4)
	if id.AuthMode != "" {
		parts = append(parts, "modo "+id.AuthMode)
	}
	if id.Issuer != "" {
		parts = append(parts, "issuer "+id.Issuer)
	}
	if id.TeamID != "" {
		parts = append(parts, "team "+id.TeamID)
	}
	parts = append(parts, "expira "+formatExpiry(id.ExpiresAt))
	return strings.Join(parts, "  ")
}

func formatExpiry(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.UTC().Format("2006-01-02 15:04") + "Z"
}
