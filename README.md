<div align="center">

# opsforge 🔥

**Your DevOps workstation, forged in minutes.**

Pick your CLIs from an interactive terminal UI, install them in one go, and turn
your zsh into a context-aware DevOps environment — live completion, a prod-aware
prompt, and **policy-as-code guards** that stop you from nuking the wrong cluster.

[![CI](https://github.com/Mrg77/opsforge/actions/workflows/ci.yml/badge.svg)](https://github.com/Mrg77/opsforge/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Mrg77/opsforge?sort=semver)](https://github.com/Mrg77/opsforge/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/Mrg77/opsforge)](https://goreportcard.com/report/github.com/Mrg77/opsforge)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

![opsforge demo](demo/demo-v0.3.2.gif)

**[Install](#install) · [Tour](#a-quick-tour) · [Shell](#the-devops-shell-environment) · [History](#history) · [Guards](#policy-as-code-guards) · [CI](#ci--machine-readable-output) · [Catalog](#the-catalog) · [Themes](#themes) · [Under the hood](#engineering-highlights)**

</div>

---

## What it is

opsforge is **three tools in one binary**:

| | | |
|:--:|---|---|
| 📦 | **Tool installer** | An interactive picker over **106 curated DevOps CLIs**. Detects what you have, what's outdated, installs the rest via Homebrew *or* direct GitHub-release binaries — works on a bare Linux box with no package manager. |
| 🐚 | **DevOps shell** | One command turns your own zsh into a Warp/Fish-like experience: a live completion menu, inline `?` help, a prod-aware prompt, and [**policy-as-code guards**](#policy-as-code-guards) on destructive commands. No shell replacement, no lock-in. |
| 📸 | **Workstation-as-code** | `opsforge snapshot` exports your whole setup — tools, profiles, shell, theme *and* guard policy — to one YAML; `opsforge apply <url>` rebuilds it anywhere, and `apply --check` verifies a machine against it in CI. Your workstation becomes a reproducible, enforceable baseline. |

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

Then keep it current with `opsforge self update` — it downloads the latest
release, **verifies its published SHA-256 before swapping the binary in place**,
and no-ops when you're already up to date (`--check` for cron/CI).

> **Windows:** use WSL — the installer backend is Homebrew and the shell layer
> targets zsh. Native winget/scoop + PowerShell support is on the roadmap.

---

## A quick tour

```sh
opsforge              # interactive picker (tabs: 1 Tools · 2 Updates · 3 Security)
opsforge status       # one-glance cockpit of your workstation
opsforge doctor       # full health check — incl. CVEs & leaked secrets
opsforge audit        # scan installed tools for CVEs (--secrets: leaked creds too)
opsforge guard test "terraform destroy" --context prod   # simulate a guard rule
opsforge apply --check team-baseline.yaml   # verify this machine matches the baseline (CI)
opsforge self update  # self-update, checksum-verified before the swap
opsforge audit --json # machine-readable output for CI (non-zero exit on HIGH/CRITICAL)
```

<table>
<tr><th align="left">Command</th><th align="left">What it does</th></tr>
<tr><td><code>opsforge</code></td><td>Interactive picker — browse, check, install</td></tr>
<tr><td><code>opsforge status</code></td><td>Cockpit: tools, updates, shell, theme at a glance</td></tr>
<tr><td><code>opsforge install kubectl helm</code></td><td>Non-interactive install by name (scriptable)</td></tr>
<tr><td><code>opsforge install --profile aws-k8s</code></td><td>Install a whole stack preset in one command</td></tr>
<tr><td><code>opsforge upgrade [-u] [tool…]</code></td><td>Upgrade all, only outdated (<code>-u</code>), or named tools</td></tr>
<tr><td><code>opsforge audit [--secrets] [--json]</code></td><td>CVE scan of installed tools · optional leaked-secrets scan · <code>--json</code> + non-zero exit gates CI</td></tr>
<tr><td><code>opsforge guard [init|list|test|lint]</code></td><td>Policy-as-code guards on destructive commands · <code>lint</code>/<code>test --json</code> make them CI-enforceable (see <a href="#policy-as-code-guards">Guards</a>)</td></tr>
<tr><td><code>opsforge use terraform@1.5</code></td><td>Pin a tool version here (delegates to mise/asdf)</td></tr>
<tr><td><code>opsforge snapshot</code> / <code>apply</code></td><td>Export / rebuild a whole workstation</td></tr>
<tr><td><code>opsforge apply --check &lt;file-or-url&gt;</code></td><td>Verify a machine against the baseline without changing it · non-zero exit on drift (<code>--json</code>)</td></tr>
<tr><td><code>opsforge self [version|update]</code></td><td>Report the version or self-update — checksum-verified before the swap (<code>--check</code> for CI/cron)</td></tr>
<tr><td><code>opsforge history [family|tool]</code></td><td>Recent shell commands, grouped by tool family (<code>kube</code>, <code>git</code>, <code>tf</code>… — see <a href="#history">History</a>)</td></tr>
<tr><td><code>opsforge list [all] [-u]</code></td><td>Installed tools · full catalog · only updates (<code>--json</code> to script)</td></tr>
<tr><td><code>opsforge profiles</code></td><td>Stack profiles with install status</td></tr>
<tr><td><code>opsforge theme [set &lt;name&gt;]</code></td><td>List/preview/persist color themes</td></tr>
<tr><td><code>opsforge doctor</code></td><td>Full health check — system, shell, toolbox, <strong>CVEs &amp; leaked secrets</strong> (<code>--json</code>)</td></tr>
</table>

> **Machine-readable everywhere.** A global `--json` flag makes `list`, `status`,
> `doctor` and `audit` emit structured JSON instead of the TUI — see
> [CI / machine-readable output](#ci--machine-readable-output).

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
opsforge snapshot -o my-setup.yaml    # tools + profiles + shell + theme + guards + version manager → one file
opsforge apply <file-or-url>          # rebuild it on any machine
opsforge apply --check <file-or-url>  # verify a machine against it, without changing a thing
```

A snapshot now captures the **whole** managed workstation — installed tools,
your custom profiles, the shell-environment state, the active **theme**, your
**guard policy** (the raw `guards.yaml`), and the detected **version manager**.
`apply` shows the full plan and asks before changing anything (`--yes` for
scripts), restoring the theme and guard rules alongside the tools. Onboarding a
new engineer becomes one command.

**A verifiable team baseline.** `apply --check` reads the snapshot and compares
this machine to it **without modifying anything**, exiting **non-zero on drift** —
a missing tool, or a theme/guards/shell/version-manager that differs. With
`--json` it emits a structured report — `{compliant, missing_tools, drift}` —
so a CI job can assert that a developer's laptop or a base image still matches
the team baseline:

```sh
opsforge apply --check team-baseline.yaml            # fails the job on any drift
opsforge apply --check team-baseline.yaml --json | jq '.compliant'
```

Snapshots are **forward-compatible**: the format grew from v1 (tools, profiles,
shell) to v2 (adds theme, guards, version manager), and older v1 files still
load — the new fields simply stay unset.

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

- **Calm, on-demand editing** — nothing pops open as you type: just a grey inline
  suggestion from your history. `↑`/`↓` search history by the **whole-line
  prefix** you've typed, `→` accepts the whole suggestion, `Tab` accepts it one
  word at a time, and the line is syntax-colored as you go. Even terraform (which
  ships no zsh completion) and opsforge itself are covered.

  <table>
  <tr><th align="left">Key</th><th align="left">What it does</th></tr>
  <tr><td><code>↑</code> / <code>↓</code></td><td>Walk history by the line prefix you've typed (<code>kubectl get pods -n s</code> + <code>↑</code> cycles only lines starting that way)</td></tr>
  <tr><td><code>→</code></td><td>Accept the whole grey suggestion</td></tr>
  <tr><td><code>Tab</code></td><td>Accept the grey suggestion one word at a time (<code>ansible-play</code> + <code>Tab</code> → <code>ansible-playbook </code>)</td></tr>
  <tr><td><code>Ctrl-Space</code></td><td>File / command completion</td></tr>
  <tr><td><code>Ctrl-R</code></td><td>Search your whole history</td></tr>
  </table>

  Prefer the old always-open live menu (zsh-autocomplete)? Set
  `OPSFORGE_AUTOMENU=1`. Disable the whole layer with `OPSFORGE_INTERACTIVE=0`.
- **`?` help** — press `?` on an empty line for a themed cheat-sheet; type
  `kubectl get ?` for that command's help, rendered under a framed header with
  `bat`-colored man syntax; type `??` to have an AI explain your last failure.
- **Context prompt** — kube `cluster:namespace` (**red on a prod-looking
  context**), cloud account, terraform workspace — each shown only when relevant.
  Plus a clean left prompt: repo-relative dir, git branch with
  dirty/ahead/behind markers, last-command duration, and a `❯` that reddens on
  failure. Reads only local git — never a cloud or cluster.
- **Policy-as-code guards** — before a destructive command (`kubectl delete`,
  `terraform destroy`, `helm uninstall`…) on a production context, opsforge can
  confirm, warn, or block — driven by [declarative rules](#policy-as-code-guards),
  not hard-coded checks. `OPSFORGE_GUARDS=0` to disable.
- **Aliases & helpers** — `k`, `tf`, `dc`, plus `kx`/`kn` to switch kube
  context/namespace (fzf picker when available). The `history` builtin is widened
  to show the last **200** lines (`history 1` for everything), and `hg <term>`
  greps your whole history — while [`opsforge history`](#history) groups it by
  DevOps tool family.
- **Integrations** — `fzf`, `zoxide`, `atuin` wired up when present.

Every module is validated with `zsh -n` in CI, so a broken script can never
reach your shell.

<table>
<tr><th align="left">Shell command</th><th align="left">What it does</th></tr>
<tr><td><code>opsforge shell install</code></td><td>Install the zsh environment into <code>~/.zshrc</code> (idempotent)</td></tr>
<tr><td><code>opsforge shell uninstall</code></td><td>Remove it cleanly (restores <code>~/.zshrc</code>)</td></tr>
<tr><td><code>opsforge shell doctor</code></td><td>Show what's provided and its state</td></tr>
<tr><td><code>opsforge shell sync</code></td><td>Refresh the shell modules <em>and</em> cached completions (run after upgrading opsforge)</td></tr>
</table>

---

## History

Your shell history is full of the exact commands you need again — buried under
everything else. `opsforge history` pulls out just one family of DevOps tools,
so you can find last week's `kubectl port-forward` without scrolling.

```sh
opsforge history              # overview: every family, with how many recent commands each has
opsforge history kube         # recent kubectl / helm / k9s / argocd… commands
opsforge history tf           # terraform / tofu / terragrunt
opsforge history terraform    # a single tool, by name
opsforge history git -n 50    # more results (0 = no cap)
opsforge history kube --json  # machine-readable
```

Built-in families group the tools you think of together — and deliberately mirror
the domains used by [guards](#policy-as-code-guards) and profiles, so `kube`,
`tf`, `cloud`… mean the same thing across the product:

<table>
<tr><th align="left">Family</th><th align="left">Tools</th></tr>
<tr><td><code>kube</code></td><td>kubectl, helm, k9s, kubectx, kustomize, stern, kubeseal, flux, argocd…</td></tr>
<tr><td><code>git</code></td><td>git, gh, glab, lazygit, tig</td></tr>
<tr><td><code>tf</code></td><td>terraform, tofu, terragrunt, tflint, terraform-docs</td></tr>
<tr><td><code>docker</code></td><td>docker, docker-compose, podman, nerdctl, colima</td></tr>
<tr><td><code>cloud</code></td><td>aws, gcloud, az, doctl, eksctl, flyctl, vercel</td></tr>
<tr><td><code>ansible</code></td><td>ansible, ansible-playbook, ansible-galaxy, ansible-vault</td></tr>
</table>

Pass a family name, or any executable to filter by that single tool. Results are
distinct commands, most-recent first, with a `×N` run count; `--limit/-n` caps
them (default 20, `0` = all) and `--json` emits them for scripts. History is
parsed **passively** — opsforge reads the file, never executes anything.

---

## Policy-as-code guards

This is the part no other tool does. Homebrew Bundle, mise, chezmoi and aqua
install your CLIs — none of them **guard how you use them**. opsforge turns the
prod-safety layer of the shell into a small policy engine: a declarative set of
rules that decides whether a destructive command should run, warn, confirm, or be
refused — based on the context you're actually in.

Rules live in a single file, `~/.config/opsforge/guards.yaml`. Each rule matches a
**command** regex and a **context** regex, and picks an action:

| Action | Effect |
|:--|:--|
| `allow` | run normally (also the result when nothing matches) |
| `warn` | print the message, then run |
| `confirm` | require typing `yes` before running |
| `deny` | block the command outright |

```yaml
# ~/.config/opsforge/guards.yaml  (first match wins)
version: 1
rules:
  - name: "confirm destructive kubectl on prod"
    match:
      command: "kubectl (delete|drain|cordon|apply|replace)"
      context: "prod|production"
    action: confirm
    message: "This changes PRODUCTION Kubernetes resources."

  - name: "never delete namespaces on prod"
    match:
      command: "kubectl delete (ns|namespace)"
      context: "prod"
    action: deny
    message: "Deleting a prod namespace is forbidden by policy."
```

```sh
opsforge guard init                                    # write a commented starter guards.yaml
opsforge guard list                                    # show the active rules (built-in or yours)
opsforge guard test "terraform destroy" --context prod # simulate: which rule fires, and the action
opsforge guard lint                                    # validate guards.yaml — non-zero exit on error
opsforge guard test "kubectl delete ns" --context prod --json  # {command, context, matched_rule, action, message}
```

**Policy you can version and enforce in CI.** Because the rules live in one file,
a team can commit `guards.yaml` to a repo and keep it honest in the pipeline:

- `opsforge guard lint` validates the active policy and **exits non-zero** when
  it's broken — a bad regex, unknown action, or wrong version fails the job
  instead of silently falling back to the default policy at runtime.
- `opsforge guard test "<cmd>" --context prod --json` emits the decision as
  `{command, context, matched_rule, action, message}`, so a pipeline can
  **assert** that, say, `terraform destroy` is `deny`ed on prod — the same
  `Evaluate` call the shell uses, so the test can't diverge from real behavior.

This is the moat, extended: the guards aren't just enforced on your machine,
they're **testable and versionable** like the rest of your infrastructure.

- **Context is read passively.** The context string is built from your kubeconfig
  `current-context`, `AWS_PROFILE`/`AWS_VAULT`, and the terraform workspace —
  opsforge **never runs `kubectl` or `gcloud`** to figure out where you are, so
  evaluating a rule can't trigger an OIDC browser login or hang on a wrapper CLI.
- **Zero-config by default.** With no `guards.yaml`, a built-in policy reproduces
  the old prod-confirm behavior exactly — upgrading changes nothing until you opt
  into custom rules.
- **Fast on the hot path.** The shell pre-filters cheaply and only calls the
  engine (`opsforge guard check`, used internally) on commands that look
  destructive, so your prompt stays instant.
- **Fails open, loudly.** A broken `guards.yaml` can never wedge your shell: the
  command runs, and the parse error is printed to stderr so you can fix your YAML.

Disable everything for one session with `OPSFORGE_GUARDS=0`.

---

## CI / machine-readable output

opsforge isn't just a pretty TUI — a global `--json` flag makes `list`, `status`,
`doctor` and `audit` emit structured JSON, so the same binary you use interactively
also drives scripts and pipelines.

```sh
opsforge audit --json | jq '.tools[] | select(.vulnerable)'   # only the affected tools
opsforge doctor --json | jq '.status'                         # "healthy" | "warnings" | "failing"
opsforge list all --json | jq '.[] | select(.outdated).name'  # tools with an update
```

The security commands also set **meaningful exit codes**, which is what turns
opsforge into a one-line gate:

- `opsforge audit` (and `--json`) exits **non-zero on any HIGH/CRITICAL CVE**.
- `opsforge audit --secrets` adds leaked credentials to the report; a **critical
  leak** exits non-zero too.
- `opsforge doctor --json` returns `{status, passed, warnings, failed, checks[]}`
  and fails when a check fails.

Ready-to-use GitHub Actions workflow: [`examples/ci-security-gate.yml`](examples/ci-security-gate.yml)
— it installs opsforge and fails the pipeline on any HIGH/CRITICAL CVE or leaked
credential, uploading the JSON reports as artifacts.

```yaml
# excerpt — audit exits non-zero on HIGH/CRITICAL, failing the job on its own
- name: CVE audit
  run: opsforge audit --json | tee cve-report.json
```

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

**Supply-chain: checksum verification.** Before a GitHub-release binary is made
executable, opsforge verifies its **SHA-256 against a published checksum** —
`checksums.txt`, `<asset>.sha256`, or a template declared per tool via the
catalog's `checksum:` field. A mismatch **refuses the install**; a release that
publishes no checksum is a warning, not a failure (best-effort, so the ~85% of
projects that ship none still install).

---

## Themes

The whole UI is themeable — one palette drives every command:

```sh
opsforge theme              # list all themes with a color preview
opsforge theme dracula      # preview one
opsforge theme set dracula  # persist it — every command follows, no reload
```

Themes: `forge` (default), `nord`, `dracula`, `gruvbox`, `light`, `mono`, `auto`.
`auto` matches your terminal background; `mono` is color-free for logs/CI. The
theme now drives **every command *and* the interactive picker** — one palette, no
stray default colors anywhere. Precedence: `$OPSFORGE_THEME` › saved (`theme
set`) › auto.

---

## Engineering highlights

The parts worth pointing a reviewer to:

- **Policy engine for the shell.** Prod guards aren't hard-coded `if`s — they're a
  declarative policy (regex × context → allow/warn/confirm/deny), first-match-wins,
  validated on load, with a behavior-preserving built-in default. Context is read
  passively (kubeconfig / env / tf workspace) so evaluation never triggers an OIDC
  login, and the shell only calls the engine on commands that look destructive.
- **CVE audit with real version matching.** Queries OSV.dev per tool, filters
  vulnerabilities *client-side* against OSV's affected ranges (semver
  `introduced`/`fixed`) and dedupes CVEs listed under multiple advisory IDs — so
  it reports only what affects the version you run, with the fix on your branch.
  Severity comes from a real **CVSS v3.1 base-score computation** off the OSV
  vector, so a critical CVE is never mis-ranked or missed.
- **Supply-chain checksum verification.** GitHub-release binaries are SHA-256
  checked against a published checksum (`checksums.txt`, `<asset>.sha256`, or a
  catalog `checksum:` template) *before* they're made executable — a mismatch
  refuses the install; a release with no checksum degrades to a warning.
- **A self-update that verifies its own integrity.** `opsforge self update`
  fetches the latest release, checks its published SHA-256, and only then
  replaces the running binary — atomically. The same supply-chain guarantee the
  installer gives your tools, opsforge applies to itself: a tampered asset is
  never made executable. `--check` reports availability with an exit code for
  cron/CI, and a dev build (no release tag to compare) is a safe no-op.
- **One source of truth for tool families.** The DevOps "families" (`kube`,
  `tf`, `cloud`…) that `history` filters by and that the guard prefilter derives
  from now live in a single package (`internal/families`) — the taxonomy that
  was once hard-coded in three diverging places. Adding a tool to a family, or a
  new danger verb, is a one-line change consumed everywhere at once.
- **Machine-readable, with exit codes that mean something.** A global `--json`
  flag renders `list`/`status`/`doctor`/`audit` as structured JSON; `audit` exits
  non-zero on HIGH/CRITICAL CVEs (and critical secret leaks with `--secrets`), so
  it drops into CI as a security gate with no wrapper script.
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
cmd/                Cobra commands (install, status, audit, guard, snapshot, apply, self, doctor, shell, theme…)
internal/catalog/   Embedded YAML catalog + brew/github validation
internal/detect/    Concurrent PATH + version detection + brew-outdated
internal/installer/ Backend router: Homebrew + GitHub-releases download (checksum.go: SHA-256 verify; self-update)
internal/audit/     OSV.dev client + client-side version matching + CVSS v3.1 scoring
internal/families/  Single source of truth for DevOps tool families (consumed by history + guard prefilter)
internal/history/   Passive shell-history reader + DevOps tool-family grouping
internal/secrets/   Leaked-credential scanner
internal/output/    Machine-readable JSON emitter for the --json flag
internal/snapshot/  Workstation capture / apply / --check drift report
internal/tui/       Bubble Tea picker with tabs (theme-bound styling)
internal/shellcfg/  zsh environment modules + completion cache + guard policy engine (policy.go)
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
