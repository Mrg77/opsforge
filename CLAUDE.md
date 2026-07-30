# opsforge

A local-first DevOps **workstation** CLI in Go (cobra). Module `github.com/Mrg77/opsforge`.

## What it is (and the moat)

opsforge sets up and hardens a DevOps engineer's own machine. It has three roles,
in order of what actually differentiates it:

1. **Guard** *(the moat — lead with this)* — policy-as-code shell guards that
   intercept destructive commands (`kubectl delete`, `terraform destroy`, `rm -rf /`…)
   by context. Rules are declarative (regex × context → allow/warn/confirm/deny),
   versionable, CI-testable. Extends to AI agents via MCP (`check_guard_policy`).
2. **Trace** — security posture of the machine: CVEs on installed tools (OSV.dev),
   leaked credentials (`verify`/`audit --secrets`), SBOM + VEX (signed), posture score.
3. **Install** *(the commodity, not the value)* — a catalog of **288** DevOps CLIs,
   installed via Homebrew or GitHub releases (works on a bare Linux box too).

It is a **personal** tool: no server, no account. When pitching, lead with the
guards, not the installer.

## Architecture

- `cmd/` — one file per cobra command (~36). Entry: `cmd/root.go`.
- `internal/` — the engine packages. Key ones:
  - `catalog/` — `catalog.yaml` is the tool catalog (288 tools + profiles). Validated in CI.
  - `shellcfg/` — the zsh/fish/bash **shell layer**: modules in `modules/*.zsh`, a
    declarative guard **policy** (`policy.go`), prompt, completions. Load order matters.
  - `envvault/` + `envcatalog/` — encrypted env-var vault (see below).
  - `i18n/` — bilingual EN/FR CLI (EN is source of truth; adding a language = one map).
  - `audit/ cve/ secrets/ sbom/ vex/ attest/ credscan/ imagescan/` — the security engine.
  - `notices/` — the cached digest (posture score lives here) shown by `status`/shell startup.
  - `mcp/` — read-only MCP server for AI agents (exposes `check_guard_policy`).

## The shell layer (the differentiator)

`opsforge shell install` writes modules to `~/.config/opsforge/shell/` and adds
`eval "$(opsforge shell env)"` to the rc file. Modules are **embedded in the binary**
via `go:embed` — so **a module change has NO effect until the binary is rebuilt**.

Load order (zsh, in `shellcfg.go` / `shell.go`):
`leftprompt → prompt → aliases → integrations → interactive → env-completion → help-completion → help → guards → notify`

Notable behaviors:
- **starship**: if installed, the prompt module initializes it and the opsforge
  prompt (left + kube-aware right) stands down. Keyed on starship's precmd **hook**
  (`prompt_starship_precmd`), NOT `$STARSHIP_SHELL` (which nested/IDE shells inherit).
  Opt out with `OPSFORGE_STARSHIP=0`.
- **`export <TAB>`** completes known DevOps env vars from `envcatalog` (even unset ones);
  never completes a secret's value.
- **Dynamic subcommand completion** (`help-completion.zsh`): for tools with no native
  zsh completer (terraform, tofu, terragrunt, packer), completes subcommands by parsing
  the tool's own `--help` on the fly, at every level. Disable: `OPSFORGE_HELPCOMP=0`.

## Encrypted env vault (`opsforge env`)

Persist secrets without cleartext. **Wrapped-key + RAM agent** (ssh-agent model):
- The vault (`~/.config/opsforge/env.age`) is encrypted to a random X25519 identity;
  that identity (`identity.age`) is itself wrapped with the passphrase (age scrypt).
- `env unlock` decrypts the key once and caches it in a **0600 RAM session file**
  (`$XDG_RUNTIME_DIR`/`$TMPDIR`, 15-min TTL). `set`/`list`/`load`/`opsenv` reuse it
  with **no re-prompt**; `env lock` clears it. What's cached is the key, never the passphrase.
- Crypto is delegated to `filippo.io/age` — **never roll our own crypto**.

## Working conventions (IMPORTANT)

- **READMEs**: `README.md` (EN) + `README.fr.md` (FR) are bilingual and must BOTH be
  updated **in the same commit** as any feature/flag/behavior change. Non-negotiable.
- **Public guide**: a separate repo `../devops-guides` hosts `opsforge-guide.html`
  (the recruiter-facing showcase, FR). It is NOT covered by the feature commit — update
  and push it **separately** in its own repo whenever a feature ships.
- **Validate end-to-end, don't assume.** Everything must be POC-able and verified by
  actually running it (build, `go test ./...`, real zsh via `expect` for shell features) —
  never "it should work". Test shell/completion in an isolated `$HOME`, never leaking
  real secrets onto the host.
- **Dev loop for shell changes**: `go build -o ~/.local/bin/opsforge . && opsforge shell sync && exec zsh`
  (the user runs opsforge as their daily shell; a stale binary means the module change isn't live).
- **CI/JSON mode**: commands support `--json` + non-zero exit codes to gate CI.
- **Delegate crypto/parsing/security to vetted libs**; opsforge orchestrates, it doesn't reinvent.

## Commit / identity

- On branch `main`, push when the work is validated.
- Public repo → **pseudonym only**: author is `Mrg77`, email is the GitHub noreply
  address (`52857557+Mrg77@users.noreply.github.com`). Never a real name or personal
  email anywhere in the repo (commits, LICENSE, READMEs, this file, generated content).
  Check `git config user.email` before committing. This is strict.
- Respond to the user in **French**.
