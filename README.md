<div align="center">

# opsforge 🔥

**Your DevOps workstation, forged in minutes.**

Pick your CLIs from an interactive terminal UI, install them in one go, and turn
your zsh into a context-aware DevOps environment — completions, a prod-aware
prompt, and guards that stop you from nuking the wrong cluster.

[![CI](https://github.com/Mrg77/opsforge/actions/workflows/ci.yml/badge.svg)](https://github.com/Mrg77/opsforge/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Mrg77/opsforge?sort=semver)](https://github.com/Mrg77/opsforge/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/Mrg77/opsforge)](https://goreportcard.com/report/github.com/Mrg77/opsforge)
[![Go Reference](https://pkg.go.dev/badge/github.com/Mrg77/opsforge.svg)](https://pkg.go.dev/github.com/Mrg77/opsforge)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

![opsforge demo](demo/opsforge.gif)

</div>

---

## What it does

opsforge is three tools in one binary:

1. **A tool installer** — an interactive picker over a curated catalog of **106
   DevOps CLIs** (Kubernetes, IaC, cloud, containers, observability, security,
   secrets, serverless…).
   It detects what you already have, what can be upgraded, and installs the rest
   via Homebrew *or* direct GitHub-release binaries (so it works on a bare Linux
   server with no package manager).

2. **A DevOps shell environment** — one command turns your own zsh into a
   Warp/Fish-like experience: a **live completion menu** as you type (no TAB),
   gray inline suggestions, syntax coloring, **inline `?` help** for any command,
   a prod-aware prompt, and **guards that make you confirm destructive commands
   on a prod cluster**.

3. **Workstation-as-code** — `opsforge snapshot` exports your whole setup
   (tools, profiles, shell state) to one YAML file; `opsforge apply <url>`
   rebuilds it on any machine. Your workstation becomes infrastructure.

No shell replacement, no lock-in: your scripts and CI keep working, and
`opsforge shell uninstall` restores everything.

## Why

Setting up (or rebuilding) a DevOps workstation means installing 20+ CLIs, then
wiring completions, aliases and a useful prompt for each — by hand, again, on
every new machine. opsforge turns that into a two-minute interactive session and
keeps your shell in sync as your toolbox evolves.

## Install

**One-liner (macOS & Linux):**

```sh
curl -fsSL https://raw.githubusercontent.com/Mrg77/opsforge/main/install.sh | sh
```

Downloads the right binary for your OS/arch from the latest GitHub release into
`~/.local/bin` (override with `OPSFORGE_INSTALL_DIR`, pin a version with
`OPSFORGE_VERSION=v1.2.3`).

**Alternatives:**

```sh
go install github.com/Mrg77/opsforge@latest   # from source
# or grab an archive from the releases page
```

> **Windows:** not supported natively (the installer backend is Homebrew and the
> shell layer targets zsh) — it works fine under WSL. Native support via
> winget/scoop + PowerShell completions is on the roadmap.

## The tool installer

Launching the bare binary opens the interactive picker — browse by category,
check what you want, hit install. It detects what you already have and what can
be upgraded, at a glance.

**Views (k9s-style tabs):** `1` Tools (full catalog) · `2` Updates (only what's
outdated) · `3` Security (live CVE scan of your installed tools).

**Keys:** `space` toggle · `u` select all updates · `a` select all
not-installed · `s` save selection as a profile · `/` filter · `i` install ·
after a run, `enter`/`m` returns to the (re-scanned) menu, `q` quits.

**At-a-glance markers:**

| Marker | Meaning |
|---|---|
| `[✓]` green | installed and up to date (shows the detected version) |
| `[✓]` orange | installed, **newer version available** (select it to upgrade) |
| `[▸]` cyan | selected for install/upgrade this run |
| `[ ]` grey | not installed |

### Commands

| Command | What it does |
|---|---|
| `opsforge` / `opsforge install` | Interactive picker: browse the catalog, check tools, install them |
| `opsforge status` | One-glance cockpit: tools, updates, shell, theme |
| `opsforge theme [name]` | List/preview color themes (`OPSFORGE_THEME` to set) |
| `opsforge install kubectl helm` | Non-interactive install by name (scriptable) |
| `opsforge install --profile aws-k8s` | Install a whole stack preset in one command |
| `opsforge profiles` | List stack profiles with installed/total counts |
| `opsforge upgrade` | Upgrade installed tools — all, `-u` for only outdated, or `upgrade jq yq gh` |
| `opsforge audit` | Scan installed tools for CVEs (`--secrets`: find leaked credentials too) |
| `opsforge explain --last` | AI-explain your last command — or just type `??` in the shell |
| `opsforge use terraform@1.5` | Pin a tool version in this dir (delegates to mise/asdf) |
| `opsforge snapshot` | Export your whole workstation setup to one shareable YAML |
| `opsforge apply <file-or-url>` | Rebuild a workstation from a snapshot (plan + confirm first) |
| `opsforge list` | Your installed tools (`list all` for the full catalog, `list -u` for updates) |
| `opsforge doctor` | Health check: brew, PATH, shell layer, version manager |

### Keeping tools current

```sh
opsforge list -u              # see what has an update
opsforge upgrade -u           # upgrade only those
opsforge upgrade jq yq gh     # or upgrade specific tools
opsforge upgrade              # upgrade everything installed
```

`update` is an alias for `upgrade`.

### Stack profiles

Install a whole stack in one command instead of picking tools one by one:

```sh
opsforge install --profile aws-k8s   # aws, eksctl, kubectl, helm, k9s, terraform, docker…
opsforge profiles                    # list all profiles with install status
```

Built-in profiles: `core`, `k8s`, `aws-k8s`, `gcp-k8s`, `iac`, `observability`,
`security`.

**Save your own.** In the picker, select the tools that make up *your* stack and
press `s` to save them as a named profile. It's written to
`~/.config/opsforge/profiles.yaml`, so you can reproduce your exact environment
on any machine:

```sh
opsforge install --profile my-stack   # reinstall your saved stack anywhere
```

### Workstation as code

Your machine setup shouldn't be a snowflake you rebuild by hand. Capture it
once, version it, reproduce it anywhere:

```sh
# on your current machine
opsforge snapshot -o my-setup.yaml        # tools + profiles + shell state
git -C ~/dotfiles add my-setup.yaml       # version it with your dotfiles

# on a brand-new machine
opsforge apply https://raw.githubusercontent.com/you/dotfiles/main/my-setup.yaml
```

`apply` shows the full plan first (what will be installed, what's already
there, anything unknown) and asks before changing a thing (`--yes` for
scripts). Perfect for new laptops, and for teams: onboarding a new engineer
becomes one command.

### Pinning tool versions

Need a specific version to reproduce or debug something (`terraform@1.5` behaves
differently than `1.6`)? opsforge delegates to a real version manager instead of
reinventing one:

```sh
opsforge install mise             # once
opsforge use terraform@1.5        # pins it in this directory
```

`opsforge use` prefers **mise** (its one-shot `mise use` installs and pins) and
falls back to **asdf**, writing the project's `mise.toml` / `.tool-versions`. It
works for any runtime those managers support, not just catalog tools.

### Security audit

```sh
opsforge audit
```

Cross-references your **installed** tool versions against the
[OSV.dev](https://osv.dev) vulnerability database and reports the tools with
known CVEs, sorted by severity, with the version that fixes each one:

```
⚠ argocd         2.11.0
    [CRITICAL] CVE-2025-47933 Argo CD allows cross-site scripting…  → fixed in 2.13.8
    [HIGH]     CVE-2025-59531 Unauthenticated argocd-server panic…  → fixed in 2.14.20
✓ helm           4.2.3 — no known vulnerabilities
```

Version matching is done client-side against OSV's affected ranges, so a CVE
fixed before your version (or only in a future major) is not reported — you see
only what actually affects the version you run.

**Leaked-credentials scan.** `opsforge audit --secrets` checks the places
secrets habitually leak on a workstation — shell history (`kubectl create
secret --from-literal`, `export GITHUB_TOKEN=…`, `docker login -p`), shell rc
files, local `.env` files — against gitleaks-style rules (AWS keys,
GitHub/GitLab/Slack tokens, private keys, JWTs…). Values are always masked;
exits non-zero on critical findings so you can wire it into CI.

### `??` — explain my last failure

Command failed? Type `??` and an AI explains what went wrong and gives the fix,
using your last command + exit code. The backend is pluggable: the `claude` CLI
if installed, else `ollama`, else any command via `OPSFORGE_AI_CMD`. Also works
directly: `opsforge explain "kubectl drain"`.

## The DevOps shell environment

```sh
opsforge shell install && exec zsh
```

This turns your **own zsh** into a DevOps-aware environment, delivered as small
modules under `~/.config/opsforge/shell/`:

- **Context prompt** — kube `cluster:namespace` (**red on a prod-looking
  context** so you notice before a mistake), active cloud account (AWS profile /
  GCP project), and terraform workspace. Each segment shows only when relevant.
- **Prod guards** — before a destructive command (`kubectl delete`,
  `terraform destroy`, `helm uninstall`…) runs against a production context,
  opsforge makes you type `yes`. Disable per-session with `OPSFORGE_GUARDS=0`.
- **Aliases & helpers** — `k`, `tf`, `dc`, plus `kx`/`kn` to switch kube
  context/namespace (fzf picker when available). All guarded on the tool being
  installed, so nothing shadows a command you don't have.
- **A clean left prompt** — current directory (repo-relative in a git repo),
  git branch with a dirty/ahead/behind marker, the duration of the last slow
  command, and a `❯` that turns red on failure. Reads only local git — never
  queries a cloud or cluster. Skips an existing prompt framework (starship,
  p10k…); force with `OPSFORGE_PROMPT=1`, disable with `=0`.
- **Live completion menu** — as you type, a menu of matching subcommands/flags
  appears automatically (no TAB), navigable with arrows; plus gray inline
  suggestions from history (→ accepts) and green/red syntax coloring. Powered by
  zsh-autocomplete / zsh-autosuggestions / zsh-syntax-highlighting, installed for
  you.
- **Inline help with `?`** — type `?` at the end of any command (e.g.
  `kubectl get ?`) to instantly see its help, without losing your line. It
  renders an elegant framed header (the command + its one-line gist) over a
  `bat`-colored body (man syntax — orange sections, green flags), and pages
  cleanly (quits if it fits, `q` to close otherwise). `bat` is installed for
  you; it falls back to a light colorizer if absent. Runs with a neutralized
  `KUBECONFIG` so it never triggers cluster auth. Disable with `OPSFORGE_HELP=0`.
- **Integrations** — `fzf`, `zoxide` and `atuin` are wired up when present, so
  history search and directory jumping just work.
- **Completions** — cached zsh completions for every installed tool (including
  opsforge itself), with static completion for tools like terraform that ship
  none.

Every module is validated with `zsh -n` in CI, so a broken script can never
reach your shell.

### Shell commands

| Command | What it does |
|---|---|
| `opsforge shell install` | Install the DevOps zsh environment into `~/.zshrc` (idempotent) |
| `opsforge shell uninstall` | Remove it cleanly (restores `~/.zshrc`, deletes modules) |
| `opsforge shell doctor` | Show what the shell environment provides and its state |
| `opsforge shell sync` | Regenerate cached zsh completions for installed tools |
| `opsforge shell env` | Print the zsh snippet (`eval "$(opsforge shell env)"`) |

## The catalog

106 curated tools across 14 categories: Kubernetes, Infrastructure as Code, Cloud
CLIs, Containers, Git & CI/CD, Observability & Monitoring, Logs, Networking &
HTTP, Databases, Security & Compliance, Secrets & Identity, Serverless & PaaS,
Runtime & Versions, Utilities. The catalog is a single embedded
[YAML file](internal/catalog/catalog.yaml) — adding a tool is a five-line PR.

### Install backends

opsforge picks a backend per tool at runtime:

- **Homebrew** (default when `brew` is on PATH) — always the latest released
  version; `opsforge upgrade` refreshes the whole toolbox.
- **GitHub releases** — for hosts without Homebrew (bare Linux servers, CI
  images), tools carrying a `github:` block (k9s, kind, kubectx, stern, argocd,
  flux, grype, syft, gitleaks, cosign, lazygit, lazydocker, zoxide, eza…) are
  installed by downloading and extracting their release binary into
  `~/.local/bin`. No package manager required.

Force a backend with `OPSFORGE_BACKEND=brew|github`; change the binary target
dir with `OPSFORGE_BIN_DIR`.

## Themes

opsforge's whole visual identity is themeable — one shared palette drives every
command. Pick one with `OPSFORGE_THEME` in your `~/.zshrc`:

```sh
opsforge theme            # list all themes with a color preview
opsforge theme dracula    # preview one
export OPSFORGE_THEME=nord # forge (default), nord, dracula, gruvbox, light, mono, or auto
```

`auto` matches your terminal background; `mono` is color-free for logs/CI.

## Engineering highlights

The parts I'd point a reviewer to:

- **CVE audit with real version matching.** `opsforge audit` queries OSV.dev per
  installed tool, then filters vulnerabilities *client-side* against OSV's
  affected ranges (semver `introduced`/`fixed`) and dedupes CVEs that appear
  under multiple advisory IDs — so it reports only what affects the version you
  actually run, with the fix version on your branch.
- **Auth-safe version detection.** Probing `kubectl --version` on a machine
  where kubectl is a cloud-SDK dispatcher wired to an OIDC plugin can pop a
  browser login. Every probe runs with a neutralized `KUBECONFIG` and a
  `WaitDelay`, so detection never triggers auth or hangs on a wrapper CLI that
  keeps the output pipe open.
- **The catalog can't lie.** A CI job validates all 68 brew references against
  the Homebrew API and every GitHub asset template against the tool's real
  latest release, across darwin/linux × amd64/arm64 — so a renamed formula or a
  wrong asset name is caught before a user hits it mid-install.
- **Homebrew edge cases handled.** Auto-taps third-party taps
  (`hashicorp/tap`, `fluxcd/tap`…) and recovers from link conflicts
  (`brew link --overwrite`) that otherwise fail a `docker` upgrade.
- **Never breaks your shell.** Shell modules are `zsh -n`-checked in CI; the
  `shell env` snippet does only PATH lookups (no subprocesses) to keep startup
  fast; slow work happens in `shell sync`.

## Architecture

```
cmd/                Cobra commands (install, list, profiles, upgrade, doctor, shell)
internal/catalog/   Embedded YAML catalog + brew/github validation
internal/detect/    Concurrent PATH + version detection + brew-outdated
internal/installer/ Backend router: Homebrew + GitHub-releases download
internal/tui/        Bubble Tea picker & install progress UI
internal/shellcfg/  zsh environment modules, completion cache, ~/.zshrc management
```

## Roadmap

- [ ] bash & fish support for the shell layer
- [ ] Native Windows support (winget/scoop + PowerShell completions)
- [ ] User config file for custom tools and profiles (`~/.config/opsforge/`)
- [ ] More `github:` templates for full brew-less coverage

## Development

```sh
go test ./...                          # unit tests
OPSFORGE_SKIP_BREW_VALIDATION=1 go test ./...   # skip the network catalog checks
go vet ./...
go build -o opsforge .
```

CI runs gofmt, vet, race-enabled tests on Linux and macOS, validates the catalog
against upstream, and cross-compiles all release targets. Releases are built and
published by GoReleaser on tag push.

## License

MIT
