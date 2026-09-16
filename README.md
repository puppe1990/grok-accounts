# grok-accounts

Account switcher for the **Grok Build CLI** (xAI's `grok` binary). It keeps credentials for several accounts as local profiles and switches the active account by swapping `~/.grok/auth.json`, with no need to log in again.

## How it works

`grok` stores your session in `~/.grok/auth.json` (or `$GROK_HOME/auth.json`) and hot-reloads that file — no CLI restart needed after a switch. `grok-accounts`:

- reads the active `auth.json` and extracts the identity (email, name, team, expiry);
- saves the file contents as a profile at `~/.grok/accounts/<alias>.json` (mode `0600`; every entry, field and its order are preserved — only whitespace surrounding the file is normalized);
- on switch, writes the chosen profile back into `auth.json` atomically (temp file + rename, `0600`), snapshotting the previously active account first.

## Install

Requires Go 1.26+ and `grok` on your PATH.

```sh
make install          # installs into $(go env GOPATH)/bin
# or
make build            # builds bin/grok-accounts
```

## Usage

```sh
grok-accounts list                    # saved profiles (* = active account)
grok-accounts current                 # account in use right now
grok-accounts add [--alias NAME]      # saves the logged-in account as a profile
grok-accounts login [--alias NAME] [--device-auth]
                                      # delegates login to grok, then saves the profile
grok-accounts switch <profile|email>  # switches the active account
grok-accounts remove <profile|email>  # deletes a saved profile
```

`profile|email`: accepts the alias (`work`), the full email, or the slug derived from the email (`dev-example-com`).

Typical flow:

```sh
grok login                    # sign in with your personal account
grok-accounts add --alias personal

grok-accounts login --alias client    # opens the browser, signs in with another account
grok-accounts switch personal         # back to the first account
```

On SSH or machines without a browser: `grok-accounts login --device-auth --alias client`.

## Layout

```
cmd/grok-accounts/main.go     wiring: resolves $GROK_HOME, runs `grok login`, os.Exit
internal/authfile/            auth.json reading/atomic writing + identity parsing
internal/store/               profiles: save, list, find, remove, alias slug
internal/cli/                 commands, injectable Env (stdout/stderr/clock/login), exit codes
```

Files on disk:

| Path                               | Contents                                                        |
| ---------------------------------- | --------------------------------------------------------------- |
| `$GROK_HOME/auth.json`             | active session (owned by `grok`; the switcher only replaces it) |
| `$GROK_HOME/accounts/<alias>.json` | account snapshots (`0600`, directory `0700`)                    |
| `$GROK_HOME`                       | `$GROK_HOME` when set, otherwise `~/.grok`                      |

Exit codes: `0` success, `1` runtime error (profile not found, not logged in, invalid JSON), `2` usage error.

## Security

- Profiles store session tokens in plain text — same protection as grok's own `auth.json`: `0600` plus full-disk encryption (FileVault).
- Never commit `~/.grok/accounts/` or share profile files.
- `remove` only deletes the local profile; the active session in `auth.json` is left untouched, and `grok logout` remains the way to revoke credentials.

## Tests and verification

Tests are written before the implementation (TDD, red → green → refactor):

```sh
pnpm install     # formatting deps (prettier)
make verify      # gofmt + prettier --check + go vet + golangci-lint + go test ./... + go build
make test
```

| Package             | What it covers                                                                                                                                                                                                   |
| ------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `internal/authfile` | parsing of the real auth.json shape (entries without email, invalid expiry, corrupt JSON), atomic writes with `0600` and no leftover temp files                                                                  |
| `internal/store`    | alias slug (including a path traversal attempt), alias conflicts between accounts, refreshing the same profile, permissions, sorted list, corrupt profiles                                                       |
| `internal/cli`      | every command with an injected `Env`: active-account marker, auto-snapshot before `switch`/`login`, no-op when switching to the already active account, `grok login` failure, `--device-auth` wiring, exit codes |

## CI and hooks

- `.github/workflows/ci.yml`: runs `pnpm install --frozen-lockfile`, installs golangci-lint and then `make verify` on pushes to `main` and on pull requests.
- `.githooks/pre-commit`: `gofmt -l` + `prettier --check` + `go test ./...` — fast checks only; lint and build belong to `make verify`/CI.

```sh
git config core.hooksPath .githooks
```

Tooling: Go 1.26 (`gofmt`, `go vet`, `golangci-lint` v2) handles the Go code; prettier 3 (pnpm 11, version pinned in `packageManager`) formats markdown/yaml/json.
