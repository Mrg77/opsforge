# opsforge 🔥

[![CI](https://github.com/Mrg77/opsforge/actions/workflows/ci.yml/badge.svg)](https://github.com/Mrg77/opsforge/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Mrg77/opsforge?sort=semver)](https://github.com/Mrg77/opsforge/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/Mrg77/opsforge)](https://goreportcard.com/report/github.com/Mrg77/opsforge)
[![Go Reference](https://pkg.go.dev/badge/github.com/Mrg77/opsforge.svg)](https://pkg.go.dev/github.com/Mrg77/opsforge)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**Forge your DevOps workstation in minutes.** Pick the CLIs you need from an
interactive terminal UI, install them in one go, and get a zsh layer with
auto-generated completions, curated aliases and a kube-aware prompt.

![opsforge demo](demo/opsforge.gif)

## Why

Setting up (or rebuilding) a DevOps workstation means installing 20+ CLIs,
then wiring completions and aliases for each one, by hand, again. `opsforge`
turns that into a two-minute interactive session — and keeps your shell in
sync as your toolbox evolves.

## Install

**One-liner (macOS & Linux):**

```sh
curl -fsSL https://raw.githubusercontent.com/Mrg77/opsforge/main/install.sh | sh
```

Downloads the right binary for your OS/arch from the latest GitHub release
into `~/.local/bin` (override with `OPSFORGE_INSTALL_DIR`, pin a version
with `OPSFORGE_VERSION=v1.2.3`).

**Alternatives:**

```sh
go install github.com/Mrg77/opsforge@latest   # from source
# or grab an archive from the releases page
```

> **Windows:** not supported natively (the installer backend is Homebrew and
> the shell layer targets zsh) — it works fine under WSL. Native support
> via winget/scoop + PowerShell completions is on the roadmap.

## Usage

Launching the bare binary opens the interactive picker — browse by
category, check what you want, hit install. It detects what you already
have and what can be upgraded, at a glance.

Keys: `space` toggle · `u` select all updates · `a` select all
not-installed · `/` filter · `i` install · after a run, `enter`/`m`
returns to the menu (re-scanned) or `q` quits.

At-a-glance markers:

| Marker | Meaning |
|---|---|
| `[✓]` green | installed and up to date (shows the detected version) |
| `[✓]` orange | installed, **newer version available** (select it to upgrade) |
| `[▸]` cyan | selected for install this run |
| `[ ]` grey | not installed |

| Command | What it does |
|---|---|
| `opsforge` / `opsforge install` | Interactive picker: browse the catalog, check tools, install them |
| `opsforge install kubectl helm` | Non-interactive install by name (scriptable) |
| `opsforge install --profile aws-k8s` | Install a whole stack preset in one command |
| `opsforge profiles` | List stack profiles with installed/total counts |
| `opsforge upgrade` | Upgrade every installed catalog tool (brew or GitHub backend) |
| `opsforge list` | Catalog with live installed/version status |
| `opsforge doctor` | Health check: brew, PATH, shell layer, broken tools |
| `opsforge shell install` | Install the DevOps zsh environment into `~/.zshrc` (idempotent) |
| `opsforge shell uninstall` | Remove it cleanly (restores `~/.zshrc`, deletes modules) |
| `opsforge shell doctor` | Show what the shell environment provides and its state |
| `opsforge shell sync` | Regenerate cached zsh completions for installed tools |
| `opsforge shell env` | Print the zsh snippet (`eval "$(opsforge shell env)"`) |

### The DevOps shell environment

`opsforge shell install` turns your **own zsh** (no shell replacement,
scripts and CI untouched) into a DevOps-aware environment, delivered as
small modules under `~/.config/opsforge/shell/`:

- **Context prompt** — kube `cluster:namespace` (**red on a prod-looking
  context** so you notice before a mistake), active cloud account
  (AWS profile / GCP project), and terraform workspace. Each segment
  shows only when relevant.
- **Prod guards** — before a destructive command (`kubectl delete`,
  `terraform destroy`, `helm uninstall`…) runs against a production
  context, opsforge asks you to type `yes`. Disable per-session with
  `OPSFORGE_GUARDS=0`.
- **Aliases & helpers** — `k`, `tf`, `dc`, plus `kx`/`kn` to switch kube
  context/namespace (fzf picker when available). All guarded on the tool
  being installed.
- **Integrations** — `fzf`, `zoxide` and `atuin` are wired up when
  present, so history search and directory jumping just work.
- **Completions** — cached zsh completions for every installed tool,
  loaded with a guarded `compinit` (no double-init slowdown).

Every module is validated with `zsh -n` in CI, so a broken script can
never reach your shell.

## The catalog

68 curated tools across 11 categories: Kubernetes, Infrastructure as Code,
Cloud CLIs, Containers, Git & CI/CD, Observability & Monitoring, Logs,
Networking & HTTP, Databases, Security & Secrets, Utilities. The catalog is
a single embedded [YAML file](internal/catalog/catalog.yaml) — adding a
tool is a five-line PR.

### Install backends

opsforge picks a backend per tool at runtime:

- **Homebrew** (default when `brew` is on PATH) — always the latest
  released version; `opsforge upgrade` refreshes the whole toolbox.
- **GitHub releases** — for hosts without Homebrew (bare Linux servers,
  CI images), tools carrying a `github:` block in the catalog (k9s, kind,
  kubectx, stern, argocd, flux, grype, syft, gitleaks, cosign, lazygit,
  lazydocker, …) are installed by downloading and extracting their
  release binary into `~/.local/bin`. No package manager required. Every
  asset template is validated against upstream releases in CI.

Force a backend with `OPSFORGE_BACKEND=brew|github`; change the binary
target dir with `OPSFORGE_BIN_DIR`.

## Architecture

```
cmd/                Cobra commands (install, list, doctor, shell)
internal/catalog/   Embedded YAML catalog + validation
internal/detect/    Concurrent PATH + version detection + brew-outdated
internal/installer/ Backend router: Homebrew + GitHub-releases download
internal/tui/       Bubble Tea picker & install progress UI
internal/shellcfg/  zsh layer generation, completion cache, ~/.zshrc management
```

Design choices worth noting:

- **Version probes are concurrent and timeout-guarded** (3s + `WaitDelay`):
  wrapper CLIs like `gcloud` spawn children that would otherwise hold the
  output pipe open forever.
- **The catalog is embedded**, so the binary is self-contained and works
  offline for everything except the installs themselves.
- **`shell env` never executes tools** (PATH lookups only) to keep shell
  startup fast; slow work happens in `shell sync`.

## Roadmap

- [ ] GitHub-releases binary backend (no Homebrew required, Linux servers)
- [ ] bash & fish support for the shell layer
- [ ] Native Windows support (winget/scoop + PowerShell completions)
- [ ] Config file to define custom tools and profiles (`~/.config/opsforge/`)

## Development

```sh
go test ./...   # unit tests
go vet ./...
go build -o opsforge .
```

CI runs gofmt, vet, race-enabled tests on Linux and macOS, and
cross-compiles all release targets. Releases are built and published by
GoReleaser on tag push.

## License

MIT
