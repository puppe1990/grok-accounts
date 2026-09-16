# grok-accounts

Switcher de contas para o **Grok Build CLI** (binário `grok`, da xAI). Salva credenciais de várias contas em perfis locais e troca a conta ativa trocando o `~/.grok/auth.json`, sem refazer login toda vez.

## Como funciona

O `grok` guarda a sessão em `~/.grok/auth.json` (ou `$GROK_HOME/auth.json`) e recarrega esse arquivo automaticamente — não precisa reiniciar o CLI depois da troca. O `grok-accounts`:

- lê o `auth.json` ativo e extrai a identidade (e-mail, nome, team, expiração);
- salva o conteúdo bruto do arquivo como um perfil em `~/.grok/accounts/<apelido>.json` (bytes preservados, permissão `0600`);
- na troca, grava o perfil escolhido no `auth.json` de forma atômica (temp file + rename, `0600`), guardando antes um snapshot da conta que estava ativa.

## Instalação

Requer Go 1.26+ e o `grok` no PATH.

```sh
make install          # instala em $(go env GOPATH)/bin
# ou
make build            # gera bin/grok-accounts
```

## Uso

```sh
grok-accounts list                    # perfis salvos (* = conta ativa)
grok-accounts current                 # conta ativa no momento
grok-accounts add [--alias NOME]      # salva a conta logada como perfil
grok-accounts login [--alias NOME] [--device-auth]
                                      # delega o login ao grok e salva o perfil
grok-accounts switch <perfil|email>   # troca a conta ativa
grok-accounts remove <perfil|email>   # remove um perfil salvo
```

`perfil|email`: aceita o apelido (`trabalho`), o e-mail completo ou o slug derivado do e-mail (`dev-example-com`).

Fluxo típico:

```sh
grok login                    # entra com a conta pessoal
grok-accounts add --alias pessoal

grok-accounts login --alias cliente   # abre o browser, entra com outra conta
grok-accounts switch pessoal          # volta pra primeira conta
```

Em SSH/máquinas sem browser: `grok-accounts login --device-auth --alias cliente`.

## Estrutura

```
cmd/grok-accounts/main.go     wiring: resolve $GROK_HOME, executa `grok login`, os.Exit
internal/authfile/            leitura/escrita atômica do auth.json + parse da identidade
internal/store/               perfis: save, list, find, remove, slug de apelido
internal/cli/                 comandos, Env injetável (stdout/stderr/relógio/login), exit codes
```

Arquivos em disco:

| Caminho                              | Conteúdo                                                       |
| ------------------------------------ | -------------------------------------------------------------- |
| `$GROK_HOME/auth.json`               | sessão ativa (gerenciado pelo `grok`; o switcher só substitui) |
| `$GROK_HOME/accounts/<apelido>.json` | snapshots das contas (`0600`, pasta `0700`)                    |
| `$GROK_HOME`                         | `$GROK_HOME` se definido, senão `~/.grok`                      |

Exit codes: `0` sucesso, `1` erro de execução (perfil não encontrado, nada logado, JSON inválido), `2` erro de uso.

## Segurança

- Os perfis guardam tokens de sessão em texto plano — mesma proteção que o `auth.json` do grok: `0600` e disco criptografado (FileVault).
- Não versione `~/.grok/accounts/` nem compartilhe os arquivos de perfil.
- `remove` apaga só o perfil local; a sessão ativa em `auth.json` não é tocada, e `grok logout` continua sendo o jeito de revogar credenciais.

## Testes e verificação

Testes escritos antes da implementação (TDD, ciclo red → green → refactor):

```sh
pnpm install     # deps de formatação (prettier)
make verify      # gofmt + prettier --check + go vet + golangci-lint + go test ./... + go build
make test
```

| Pacote              | O que cobre                                                                                                                                                                                            |
| ------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `internal/authfile` | parse do formato real do auth.json (inclui entradas sem e-mail, expiry inválido, JSON corrompido), escrita atômica com `0600` e sem arquivos temporários órfãos                                        |
| `internal/store`    | slug de apelido (inclui tentativa de path traversal), conflito de apelido entre contas, refresh do mesmo perfil, permissões, list ordenado, perfis corrompidos                                         |
| `internal/cli`      | cada comando com `Env` injetado: marcador de conta ativa, auto-snapshot antes de `switch`/`login`, no-op ao trocar para a conta já ativa, falha do `grok login`, wiring do `--device-auth`, exit codes |

## CI e hooks

- `.github/workflows/ci.yml`: `pnpm install --frozen-lockfile`, instala o golangci-lint e roda `make verify` em push para `main` e em pull requests.
- `.githooks/pre-commit`: `gofmt -l` + `prettier --check` + `go test ./...` — só os checks rápidos; lint e build ficam no `make verify`/CI.

```sh
git config core.hooksPath .githooks
```

Ferramentas: Go 1.26 (`gofmt`, `go vet`, `golangci-lint` v2) cuida do código Go; prettier 3 (pnpm 11, versão pinada em `packageManager`) formata markdown/yaml/json.
