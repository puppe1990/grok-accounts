package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/puppe1990/grok-accounts/internal/store"
)

const usage = `grok-accounts — troca de contas do Grok Build CLI

uso:
  grok-accounts list                    lista os perfis salvos
  grok-accounts current                 mostra a conta ativa
  grok-accounts add [--alias NOME]      salva a conta ativa como um perfil
  grok-accounts login [--alias NOME] [--device-auth]
                      [--incognito [--browser NOME]]
                                        faz login no grok e salva o perfil
                                        --incognito abre o login em janela anônima (macOS)
  grok-accounts switch <perfil|email>   troca a conta ativa
  grok-accounts remove <perfil|email>   remove um perfil salvo
`

type LoginOptions struct {
	DeviceAuth bool
	Incognito  bool
	Browser    string
}

type Env struct {
	GrokHome string
	Stdout   io.Writer
	Stderr   io.Writer
	Now      func() time.Time
	RunLogin func(opts LoginOptions) error
}

func Run(args []string, env Env) int {
	if env.Stdout == nil {
		env.Stdout = os.Stdout
	}
	if env.Stderr == nil {
		env.Stderr = os.Stderr
	}

	if len(args) == 0 {
		fmt.Fprint(env.Stdout, usage)
		return 0
	}

	switch args[0] {
	case "list":
		return runList(args[1:], env)
	case "current":
		return runCurrent(args[1:], env)
	case "add":
		return runAdd(args[1:], env)
	case "login":
		return runLogin(args[1:], env)
	case "switch":
		return runSwitch(args[1:], env)
	case "remove":
		return runRemove(args[1:], env)
	case "help", "-h", "--help":
		fmt.Fprint(env.Stdout, usage)
		return 0
	default:
		fmt.Fprintf(env.Stderr, "comando desconhecido: %q\n\n%s", args[0], usage)
		return 2
	}
}

func ResolveGrokHome(getenv func(string) string) (string, error) {
	if home := getenv("GROK_HOME"); home != "" {
		return home, nil
	}
	home := getenv("HOME")
	if home == "" {
		return "", fmt.Errorf("HOME não definido e GROK_HOME vazio")
	}
	return filepath.Join(home, ".grok"), nil
}

func (env Env) now() time.Time {
	if env.Now == nil {
		return time.Now()
	}
	return env.Now()
}

func (env Env) store() *store.Store {
	return &store.Store{Dir: filepath.Join(env.GrokHome, "accounts"), Now: env.Now}
}

func (env Env) authPath() string {
	return filepath.Join(env.GrokHome, "auth.json")
}
