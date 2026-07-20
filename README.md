# opsforge 🔥

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
category, check what you want, hit install.

| Command | What it does |
|---|---|
| `opsforge` / `opsforge install` | Interactive picker: browse the catalog, check tools, install them |
| `opsforge install kubectl helm` | Non-interactive install by name (scriptable) |
| `opsforge install --profile aws-k8s` | Install a whole stack preset in one command |
| `opsforge profiles` | List stack profiles with installed/total counts |
| `opsforge upgrade` | Upgrade every installed catalog tool via Homebrew |
| `opsforge list` | Catalog with live installed/version status |
| `opsforge doctor` | Health check: brew, PATH, shell layer, broken tools |
| `opsforge shell install` | Adds the opsforge layer to your `~/.zshrc` (idempotent) |
| `opsforge shell sync` | Regenerates cached zsh completions for installed tools |
| `opsforge shell env` | Prints the zsh snippet (`eval "$(opsforge shell env)"`) |

### The shell layer

- **Completions** — every installed tool that ships a zsh completion script
  gets it generated and cached in `~/.cache/opsforge/completions/`, loaded at
  shell startup with a guarded `compinit` (no double-init slowdown).
- **Aliases** — a deliberately short list of muscle-memory shortcuts
  (`k`=kubectl, `tf`=terraform, `dc`=docker compose), enabled only when the
  underlying tool is installed.
- **Prompt** — current Kubernetes context (`⎈ prod-cluster`) in the right
  prompt, only if you haven't already claimed `RPROMPT`.

## The catalog

65 curated tools across 11 categories: Kubernetes, Infrastructure as Code,
Cloud CLIs, Containers, Git & CI/CD, Observability & Monitoring, Logs,
Networking & HTTP, Databases, Security & Secrets, Utilities. The catalog is
a single embedded [YAML file](internal/catalog/catalog.yaml) — adding a
tool is a five-line PR.

Everything installs through Homebrew, so you always get the latest
released version, and `opsforge upgrade` refreshes the whole toolbox in
one command.

## Architecture

```
cmd/                Cobra commands (install, list, doctor, shell)
internal/catalog/   Embedded YAML catalog + validation
internal/detect/    Concurrent PATH + version detection (timeout-guarded)
internal/installer/ Homebrew backend
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
