<div align="center">

# opsforge 🔥

**Your DevOps workstation, forged in minutes.**

Pick your CLIs from an interactive terminal UI, install them in one go, and turn
your zsh into a context-aware DevOps environment — live completion, a prod-aware
prompt, and guards that stop you from nuking the wrong cluster.

[![CI](https://github.com/Mrg77/opsforge/actions/workflows/ci.yml/badge.svg)](https://github.com/Mrg77/opsforge/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Mrg77/opsforge?sort=semver)](https://github.com/Mrg77/opsforge/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/Mrg77/opsforge)](https://goreportcard.com/report/github.com/Mrg77/opsforge)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

![opsforge demo](demo/demo-v0.3.2.gif)

**[Install](#install) · [Tour](#a-quick-tour) · [Shell](#the-devops-shell-environment) · [Catalog](#the-catalog) · [Themes](#themes) · [Under the hood](#engineering-highlights)**

</div>

---

## What it is

opsforge is **three tools in one binary**:

| | | |
|:--:|---|---|
| 📦 | **Tool installer** | An interactive picker over **106 curated DevOps CLIs**. Detects what you have, what's outdated, installs the rest via Homebrew *or* direct GitHub-release binaries — works on a bare Linux box with no package manager. |
| 🐚 | **DevOps shell** | One command turns your own zsh into a Warp/Fish-like experience: a live completion menu, inline `?` help, a prod-aware prompt, and guards on destructive commands. No shell replacement, no lock-in. |
| 📸 | **Workstation-as-code** | `opsforge snapshot` exports your whole setup to one YAML; `opsforge apply <url>` rebuilds it anywhere. Your machine becomes reproducible infrastructure. |

> **Why:** setting up (or rebuilding) a DevOps workstation means installing 20+
> CLIs, then wiring completions, aliases and a useful prompt for each — by hand,
> again, on every new machine. opsforge makes it a two-minute session and keeps
> your shell in sync as your toolbox grows.

---

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/Mrg77/opsforge/main/install.sh | sh
```

Downloads the right binary for your OS/arch into `~/.local/bin` (override with
`OPSFORGE_INSTALL_DIR`, pin with `OPSFORGE_VERSION=v1.2.3`).

```sh
go install github.com/Mrg77/opsforge@latest   # from source
```

> **Windows:** use WSL — the installer backend is Homebrew and the shell layer
> targets zsh. Native winget/scoop + PowerShell support is on the roadmap.

---

## A quick tour

```sh
opsforge              # interactive picker (tabs: 1 Tools · 2 Updates · 3 Security)
opsforge status       # one-glance cockpit of your workstation
opsforge doctor       # full health check — incl. CVEs & leaked secrets
opsforge audit        # scan installed tools for CVEs (--secrets: leaked creds too)
```

<table>
<tr><th align="left">Command</th><th align="left">What it does</th></tr>
<tr><td><code>opsforge</code></td><td>Interactive picker — browse, check, install</td></tr>
<tr><td><code>opsforge status</code></td><td>Cockpit: tools, updates, shell, theme at a glance</td></tr>
<tr><td><code>opsforge install kubectl helm</code></td><td>Non-interactive install by name (scriptable)</td></tr>
<tr><td><code>opsforge install --profile aws-k8s</code></td><td>Install a whole stack preset in one command</td></tr>
<tr><td><code>opsforge upgrade [-u] [tool…]</code></td><td>Upgrade all, only outdated (<code>-u</code>), or named tools</td></tr>
<tr><td><code>opsforge audit [--secrets]</code></td><td>CVE scan of installed tools · optional leaked-secrets scan</td></tr>
<tr><td><code>opsforge use terraform@1.5</code></td><td>Pin a tool version here (delegates to mise/asdf)</td></tr>
<tr><td><code>opsforge snapshot</code> / <code>apply</code></td><td>Export / rebuild a whole workstation</td></tr>
<tr><td><code>opsforge list [all] [-u]</code></td><td>Installed tools · full catalog · only updates</td></tr>
<tr><td><code>opsforge profiles</code></td><td>Stack profiles with install status</td></tr>
<tr><td><code>opsforge theme [set &lt;name&gt;]</code></td><td>List/preview/persist color themes</td></tr>
<tr><td><code>opsforge doctor</code></td><td>Full health check — system, shell, toolbox, <strong>CVEs &amp; leaked secrets</strong></td></tr>
</table>

### The picker

Launch the bare binary to browse by category and install what you check.

- **Tabs (k9s-style):** `1` Tools · `2` Updates (only outdated) · `3` Security
  (live CVE scan of installed tools)
- **Keys:** `space` toggle · `u` all updates · `a` all not-installed · `s` save
  selection as a profile · `/` filter · `i` install · `q` quit
- **Markers:** `[✓]` green installed · `[✓]` orange update available · `[▸]` cyan
  selected · `[ ]` grey not installed

### Stack profiles

Install a whole stack in one command — or save your own:

```sh
opsforge install --profile aws-k8s   # aws, eksctl, kubectl, helm, k9s, terraform…
opsforge profiles                    # list all with install status
```

Built-in: `core`, `k8s`, `aws-k8s`, `gcp-k8s`, `iac`, `observability`,
`security`. In the picker, select your tools and press `s` to save a personal
profile to `~/.config/opsforge/profiles.yaml` — then
`opsforge install --profile my-stack` reproduces it anywhere.

### Workstation as code

Your machine setup shouldn't be a snowflake you rebuild by hand:

```sh
opsforge snapshot -o my-setup.yaml    # tools + profiles + shell state → one file
opsforge apply <file-or-url>          # rebuild it on any machine
```

`apply` shows the full plan and asks before changing anything (`--yes` for
scripts). Onboarding a new engineer becomes one command.

### Security audit

```sh
opsforge audit             # CVEs in your installed tools
opsforge audit --secrets   # + credentials leaking in history / rc / .env
```

Cross-references installed versions against [OSV.dev](https://osv.dev), sorted by
severity, with the fix version:

```
⚠ argocd         2.11.0
    [CRITICAL] CVE-2025-47933 Argo CD allows cross-site scripting…  → fixed in 2.13.8
    [HIGH]     CVE-2025-59531 Unauthenticated argocd-server panic…  → fixed in 2.14.20
✓ helm           4.2.3 — no known vulnerabilities
```

Matching is client-side against OSV's affected ranges, so a CVE fixed before
your version (or only in a future major) isn't reported. `--secrets` scans shell
history, rc files and local `.env`s for AWS/GitHub/GitLab/Slack tokens, private
keys, `--from-literal`, `docker login -p`… with values always masked.

### Pinning tool versions

```sh
opsforge install mise
opsforge use terraform@1.5   # pins it in this directory
```

Delegates to **mise** (preferred) or **asdf** — no version-manager reinvention.

---

## The DevOps shell environment

```sh
opsforge shell install && exec zsh
```

Turns your **own zsh** into a DevOps-aware environment (modules under
`~/.config/opsforge/shell/`, `shell uninstall` restores everything):

- **Live completion menu** — a menu of matching subcommands/flags opens as you
  type (no TAB), navigable with arrows; plus grey inline history suggestions
  (→ accepts) and syntax coloring. Even terraform (which ships no zsh completion)
  and opsforge itself are covered.
- **`?` help** — press `?` on an empty line for a themed cheat-sheet; type
  `kubectl get ?` for that command's help, rendered under a framed header with
  `bat`-colored man syntax; type `??` to have an AI explain your last failure.
- **Context prompt** — kube `cluster:namespace` (**red on a prod-looking
  context**), cloud account, terraform workspace — each shown only when relevant.
  Plus a clean left prompt: repo-relative dir, git branch with
  dirty/ahead/behind markers, last-command duration, and a `❯` that reddens on
  failure. Reads only local git — never a cloud or cluster.
- **Prod guards** — before a destructive command (`kubectl delete`,
  `terraform destroy`, `helm uninstall`…) on a production context, opsforge makes
  you type `yes`. `OPSFORGE_GUARDS=0` to disable.
- **Aliases & helpers** — `k`, `tf`, `dc`, plus `kx`/`kn` to switch kube
  context/namespace (fzf picker when available).
- **Integrations** — `fzf`, `zoxide`, `atuin` wired up when present.

Every module is validated with `zsh -n` in CI, so a broken script can never
reach your shell.

<table>
<tr><th align="left">Shell command</th><th align="left">What it does</th></tr>
<tr><td><code>opsforge shell install</code></td><td>Install the zsh environment into <code>~/.zshrc</code> (idempotent)</td></tr>
<tr><td><code>opsforge shell uninstall</code></td><td>Remove it cleanly (restores <code>~/.zshrc</code>)</td></tr>
<tr><td><code>opsforge shell doctor</code></td><td>Show what's provided and its state</td></tr>
<tr><td><code>opsforge shell sync</code></td><td>Regenerate cached completions</td></tr>
</table>

---

## The catalog

**106 tools across 14 categories** — Kubernetes, Infrastructure as Code, Cloud
CLIs, Containers, Git & CI/CD, Observability & Monitoring, Logs, Networking &
HTTP, Databases, Security & Compliance, Secrets & Identity, Serverless & PaaS,
Runtime & Versions, Utilities. It's a single embedded
[YAML file](internal/catalog/catalog.yaml) — adding a tool is a five-line PR.

**Two install backends, picked per tool at runtime:**

- **Homebrew** (when on PATH) — always the latest release; `opsforge upgrade`
  refreshes the whole toolbox.
- **GitHub releases** — for hosts without Homebrew (bare Linux, CI images), tools
  with a `github:` block are installed by downloading their release binary into
  `~/.local/bin`. No package manager required.

Force one with `OPSFORGE_BACKEND=brew|github`; set the target dir with
`OPSFORGE_BIN_DIR`.

---

## Themes

The whole UI is themeable — one palette drives every command:

```sh
opsforge theme              # list all themes with a color preview
opsforge theme dracula      # preview one
opsforge theme set dracula  # persist it — every command follows, no reload
```

Themes: `forge` (default), `nord`, `dracula`, `gruvbox`, `light`, `mono`, `auto`.
`auto` matches your terminal background; `mono` is color-free for logs/CI.
Precedence: `$OPSFORGE_THEME` › saved (`theme set`) › auto.

---

## Engineering highlights

The parts worth pointing a reviewer to:

- **CVE audit with real version matching.** Queries OSV.dev per tool, filters
  vulnerabilities *client-side* against OSV's affected ranges (semver
  `introduced`/`fixed`) and dedupes CVEs listed under multiple advisory IDs — so
  it reports only what affects the version you run, with the fix on your branch.
- **Auth-safe detection.** Probing `kubectl --version` where kubectl is a
  cloud-SDK dispatcher wired to an OIDC plugin can pop a browser login. Every
  probe runs with a neutralized `KUBECONFIG` and a `WaitDelay`, so detection
  never triggers auth or hangs on a wrapper CLI holding the output pipe.
- **The catalog can't lie.** A CI job validates all 106 brew references against
  the Homebrew API and every GitHub asset template against the tool's real latest
  release (darwin/linux × amd64/arm64) — a renamed formula is caught before a
  user hits it mid-install.
- **Homebrew edge cases handled.** Auto-taps third-party taps and recovers from
  link conflicts (`brew link --overwrite`) that otherwise fail a docker upgrade.
- **Never breaks your shell.** Modules are `zsh -n`-checked in CI; the `shell
  env` snippet does only PATH lookups (no subprocesses) for fast startup.

### Architecture

```
cmd/                Cobra commands (install, status, audit, snapshot, doctor, shell, theme…)
internal/catalog/   Embedded YAML catalog + brew/github validation
internal/detect/    Concurrent PATH + version detection + brew-outdated
internal/installer/ Backend router: Homebrew + GitHub-releases download
internal/audit/     OSV.dev client + client-side version matching
internal/secrets/   Leaked-credential scanner
internal/snapshot/  Workstation capture / apply
internal/tui/       Bubble Tea picker with tabs
internal/shellcfg/  zsh environment modules + completion cache
internal/ui/        Shared visual identity + themes
```

---

## Development

```sh
go test ./...                                   # unit tests
OPSFORGE_SKIP_BREW_VALIDATION=1 go test ./...   # skip the network catalog checks
go build -o opsforge .
```

CI runs gofmt, vet, race tests on Linux & macOS, validates the catalog against
upstream, and cross-compiles all targets. Releases are cut by GoReleaser on tag.

## Roadmap

- [ ] bash & fish support for the shell layer
- [ ] Native Windows (winget/scoop + PowerShell completions)
- [ ] User config file for custom tools and profiles
- [ ] More `github:` templates for full brew-less coverage

## License

MIT
