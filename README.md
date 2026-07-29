<div align="center">

# opsforge 🔥

**A safety layer for your DevOps workstation — so a distracted command (or an AI agent) can't nuke the wrong cluster.**

You know the feeling: you meant to run that `kubectl delete` on staging, but your
shell was pointed at prod. opsforge is the seatbelt for that moment. It turns your
shell (zsh, fish or bash) into one that knows which cluster you're on, and stops a
destructive command before it lands — using rules *you* wrote, in a file. The same
rules apply whether it's you at the keyboard or [an AI agent](#ai-agents-mcp)
working on your machine.

That's the heart of it. Around it, opsforge also keeps the machine honest — it
audits [the credentials](#credential-hygiene-verify) sitting on your disk and the
[known holes](#sbom--supply-chain) in your tools — and, since none of that helps on
an empty machine, it installs and maintains your DevOps toolbox too. All in one
binary. It's a personal tool, not a team platform: no server, no account, nothing
to lock you in.

**English** · [Français](README.fr.md)

[![CI](https://github.com/Mrg77/opsforge/actions/workflows/ci.yml/badge.svg)](https://github.com/Mrg77/opsforge/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Mrg77/opsforge?sort=semver)](https://github.com/Mrg77/opsforge/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/Mrg77/opsforge)](https://goreportcard.com/report/github.com/Mrg77/opsforge)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
<br>
[![Tools](https://img.shields.io/badge/tools-288-blue)](#the-catalog)
[![SBOM](https://img.shields.io/badge/SBOM-CycloneDX%201.6-orange)](#sbom--supply-chain)
[![Made with Go](https://img.shields.io/badge/made%20with-Go-00ADD8?logo=go&logoColor=white)](https://go.dev)

![opsforge demo](demo/demo.gif)

**[Try it](#try-it-in-a-sandbox) · [Install](#install) · [Tour](#a-quick-tour) · [Workflows](#common-workflows) · [Shell](#the-devops-shell-environment) · [Guards](#policy-as-code-guards) · [Project mode](#project-mode) · [SBOM & VEX](#sbom--supply-chain) · [Verify](#credential-hygiene-verify) · [AI agents (MCP)](#ai-agents-mcp) · [CI](#ci--integrations) · [Catalog](#the-catalog) · [Under the hood](#engineering-highlights)**

</div>

---

## What it is

Think of opsforge as three tools that happen to share one binary. Here they are,
most interesting first:

| | | |
|:--:|---|---|
| 🛡️ | **The guards — the real point** | One command turns your shell (zsh, fish or bash) into one that *knows where it is*. When you type `kubectl delete` or `terraform destroy` on a production cluster, it stops you and asks first. The rules are yours, written in a file — and they protect you at the keyboard *and* [any AI agent](#ai-agents-mcp) working on your machine. Everything that trips a guard on prod is [written down](#an-audit-trail-what-did-i-run-on-prod-this-week), so you can look back later. |
| 🔐 | **Keeping the machine honest** | Two questions, answered. *Are the credentials on my machine a liability?* — [`opsforge verify`](#credential-hygiene-verify) finds keys that never expire, secrets in clear text, files anyone can read. *Do my tools have known holes?* — [`opsforge audit`](#ci--integrations) and friends produce a signed inventory ([SBOM](#sbom--supply-chain) + [VEX](#sbom--supply-chain)) of every CVE, and flag the ones attackers are *actually* exploiting. All read-only, no cloud. |
| 📦 | **Setting up the toolbox** | And because a safety layer is no use on an empty machine, opsforge also installs and keeps your DevOps tools current — a searchable picker over **288 curated CLIs**, in one binary, on macOS or a bare Linux box. Plus [reproducible setups](#project-mode): describe a machine (or a repo's toolchain) in a file and rebuild it anywhere. |

### Why it exists

A DevOps machine has a safety problem that the usual tools just… ignore.

- **A distracted `kubectl delete` on the wrong cluster** has no seatbelt. The tool
  runs it whether you're on staging or prod — it can't tell the difference, and it
  won't ask. Worse: an **AI agent** can now run it *for* you, faster, with nobody
  watching.
- **Nobody really knows what's risky on the box.** Which credentials never expire?
  Which sit in plain text? Which tool carries a hole someone is exploiting right
  now? You'd have to check by hand, tool by tool.
- **Rebuilding a workstation is a lost day.** Install twenty-something CLIs, then
  wire up completions, aliases and a decent prompt for each — every time.

opsforge rolls those into one binary because they share the same home (your shell)
and the same data (the tools it detects). It's deliberately a **personal tool, not
a team platform** — no server, no account, nothing to lock you in — so it stays
something you *run*, not something you have to *operate*. (Wondering about a whole
fleet? See [Guards at team scale](#guards-at-team-scale).)

---

## Try it in a sandbox

Want to see the guards fire without installing anything or touching real infra?
There's a throwaway demo image for exactly that. It's a forged zsh shell already
sitting in a **fake prod context**, with no-op `kubectl`/`terraform`/`helm`
stubs standing in for the real tools:

```sh
docker run --rm -it ghcr.io/mrg77/opsforge-demo
```

It opens a short guided tour (status, then the **guards**, the guard **audit
trail**, credential-hygiene **`verify`**, and the SBOM) and then drops you into
the shell. Try typing `kubectl delete namespace payments` yourself and watch the
prod guard step in. Nothing here can reach a real cluster. The "prod" context is
a one-line fake kubeconfig that opsforge only reads, the tools are stubs, and the
credentials `verify` flags are **fake** ones seeded into the image (an AWS key, a
passphrase-less SSH key, a base64 docker login). So you get to see the whole
security layer at work without a single real secret in play.

Prefer the browser? Open it in a Codespace — same image, nothing to install
locally:

[![Open in GitHub Codespaces](https://github.com/codespaces/badge.svg)](https://codespaces.new/Mrg77/opsforge?quickstart=1)

---

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/Mrg77/opsforge/main/install.sh | sh
```

That grabs the right binary for your OS and architecture and drops it in
`~/.local/bin` (override the location with `OPSFORGE_INSTALL_DIR`, or pin a
version with `OPSFORGE_VERSION=v1.2.3`). Prefer building from source?
`go install github.com/Mrg77/opsforge@latest`.

To stay current, run `opsforge self update`. It downloads the latest release,
**checks its published SHA-256 before swapping the binary in place**, and does
nothing when you're already up to date (add `--check` for cron or CI).

> **Windows:** use WSL for now. The installer leans on Homebrew and the shell
> layer targets zsh, fish and bash. Native winget/scoop and PowerShell support
> are on the roadmap.

---

## A quick tour

```sh
opsforge              # interactive picker (tabs: 1 Tools · 2 Updates · 3 Security)
opsforge status       # one-glance cockpit of your workstation
opsforge doctor       # full health check — incl. CVEs & leaked secrets
opsforge audit        # scan installed tools for CVEs (--secrets: leaked creds too)
opsforge guard test "terraform destroy" --context prod   # simulate a guard rule
opsforge apply --check my-setup.yaml                     # verify this machine matches your snapshot (CI)
opsforge self update  # self-update, checksum-verified before the swap
```

<table>
<tr><th align="left">Command</th><th align="left">What it does</th></tr>
<tr><td><code>opsforge</code></td><td>Interactive picker — browse, check, install</td></tr>
<tr><td><code>opsforge status</code></td><td>Cockpit: tools, updates, shell, theme, and a <strong>0–100 security-posture score</strong> at a glance</td></tr>
<tr><td><code>opsforge notify [--json]</code></td><td>One digest of what needs attention — CVEs, updates, leaked secrets, a newer opsforge (see <a href="#the-notify-digest">notify</a>)</td></tr>
<tr><td><code>opsforge install kubectl helm</code></td><td>Non-interactive install by name (scriptable)</td></tr>
<tr><td><code>opsforge install --profile aws-k8s</code></td><td>Install a whole stack preset in one command</td></tr>
<tr><td><code>opsforge upgrade [-u] [tool…]</code></td><td>Upgrade all, only outdated (<code>-u</code>), or named tools</td></tr>
<tr><td><code>opsforge audit [--secrets] [--json]</code></td><td>CVE scan of installed tools · optional leaked-secrets scan · <code>--json</code> + non-zero exit gates CI</td></tr>
<tr><td><code>opsforge verify [--strict] [--json]</code></td><td>Credential-hygiene audit of the workstation — static keys, clear-text secrets, loose file perms, expiring certs · read-only, offline (see <a href="#credential-hygiene-verify">verify</a>)</td></tr>
<tr><td><code>opsforge guard [init|list|test|lint|log]</code></td><td>Policy-as-code guards on destructive commands · <code>lint</code>/<code>test --json</code> make them CI-checkable (see <a href="#policy-as-code-guards">Guards</a>)</td></tr>
<tr><td><code>opsforge env [set|list|rm|load]</code></td><td>Encrypted (age) env-var vault — persist secrets without cleartext; <code>opsenv</code> unlocks them into your shell (see <a href="#encrypted-env-vault">env vault</a>)</td></tr>
<tr><td><code>opsforge use terraform@1.5</code></td><td>Pin a tool version here (delegates to mise/asdf)</td></tr>
<tr><td><code>opsforge sync [--check] [--init]</code></td><td>Install the tools a committed <code>opsforge.yaml</code> declares · <code>--check</code> reports drift for CI · optional CVE gate (see <a href="#project-mode">Project mode</a>)</td></tr>
<tr><td><code>opsforge sbom [--audit] [--sign]</code></td><td>Emit a CycloneDX 1.6 SBOM of installed tools · <code>--audit</code> embeds their CVEs · <code>--sign</code> adds a Sigstore bundle (see <a href="#sbom--supply-chain">SBOM</a>)</td></tr>
<tr><td><code>opsforge vex [--kev] [--sign]</code></td><td>Emit an OpenVEX document of the CVEs on your tools · <code>--kev</code> flags the actively-exploited (CISA KEV) ones · <code>--sign</code> signs it (see <a href="#vex--cisa-kev">VEX</a>)</td></tr>
<tr><td><code>opsforge scan &lt;image&gt; [--diff]</code></td><td>Scan a container image for CVEs (via syft/trivy + opsforge's OSV engine) · <code>--diff</code> correlates it with your workstation (see <a href="#scanning-a-container-image">scan</a>)</td></tr>
<tr><td><code>opsforge mcp</code></td><td>Run a read-only MCP server so an AI agent can query your workstation (see <a href="#ai-agents-mcp">MCP</a>)</td></tr>
<tr><td><code>opsforge snapshot</code> / <code>apply</code></td><td>Export / rebuild a whole workstation</td></tr>
<tr><td><code>opsforge apply --check &lt;file-or-url&gt;</code></td><td>Verify a machine against your snapshot without changing it · non-zero exit on drift (<code>--json</code>)</td></tr>
<tr><td><code>opsforge version</code></td><td>Print the version, commit and build info (also <code>--version</code>; <code>--json</code> to script)</td></tr>
<tr><td><code>opsforge self [version|update]</code></td><td>Report the version or self-update — checksum-verified before the swap (<code>--check</code> for CI/cron)</td></tr>
<tr><td><code>opsforge history [family|tool]</code></td><td>Recent shell commands, grouped by tool family (<code>kube</code>, <code>git</code>, <code>tf</code>… — see <a href="#history">History</a>)</td></tr>
<tr><td><code>opsforge ai</code></td><td>Show which AI backend opsforge drives (an AI CLI you already have — no key to manage), or how to set one up</td></tr>
<tr><td><code>opsforge explain [--last] &lt;cmd&gt;</code></td><td>Ask your AI to explain a command or your last failure (the shell <code>??</code> shortcut)</td></tr>
<tr><td><code>opsforge advise</code></td><td>AI-prioritized remediation plan from your real findings — what to fix first (CVEs, updates, leaked secrets) and the exact command</td></tr>
<tr><td><code>opsforge list [all] [-u]</code></td><td>Installed tools · full catalog · only updates (<code>--json</code> to script)</td></tr>
<tr><td><code>opsforge list &lt;term&gt;</code></td><td>Search the whole catalog by name, description or category (e.g. <code>list dns</code>)</td></tr>
<tr><td><code>opsforge profiles</code></td><td>Stack profiles with install status</td></tr>
<tr><td><code>opsforge theme [set &lt;name&gt;]</code></td><td>List/preview/persist color themes</td></tr>
<tr><td><code>opsforge doctor</code></td><td>Full health check — system, shell, toolbox, <strong>CVEs &amp; leaked secrets</strong> (<code>--json</code>)</td></tr>
</table>

> **Machine-readable everywhere.** A global `--json` flag makes `list`, `status`,
> `doctor` and `audit` print structured JSON instead of the TUI, so the same
> commands feed a script just as happily. See
> [CI & integrations](#ci--integrations).

### The picker

Launch the bare binary and you get an interactive picker. Browse by category and
install whatever you check off.

- **Tabs (k9s-style):** `1` Tools · `2` Updates (only outdated) · `3` Security
  (live CVE scan of installed tools)
- **Keys:** `space` toggle · `u` all updates · `a` all not-installed · `s` save
  selection as a profile · `/` filter · `i` install · `q` quit
- **Markers:** `[✓]` green installed · `[✓]` orange update available · `[▸]` cyan
  selected · `[ ]` grey not installed

---

## Common workflows

Three everyday paths that show how the pieces fit together.

### Set up a new machine

Switching laptops? Rebuild your whole workstation from a single file instead of
losing a day to manual setup.

```sh
opsforge snapshot -o my-setup.yaml         # on your current machine: tools + shell + theme + guards → one YAML
opsforge apply https://…/my-setup.yaml     # on the new one: review the plan, then rebuild everything
opsforge shell install && exec zsh         # light up the DevOps shell
```

### Gate your CI on CVEs & secrets

The same binary you use interactively doubles as a one-line security gate in a
pipeline.

```sh
opsforge audit --json | tee cve-report.json   # non-zero exit on any HIGH/CRITICAL CVE — fails the job on its own
opsforge audit --secrets --json               # also fails on a leaked credential
```

There's a drop-in workflow ready to copy: [`examples/ci-security-gate.yml`](examples/ci-security-gate.yml).

### Version & validate your prod-guard policy

Your prod-safety rules live in one file, so you can version them and keep them
honest in the pipeline, the same way you'd version the rest of your dotfiles.

```sh
opsforge guard init                                            # write a starter guards.yaml, then commit it
opsforge guard lint                                            # validate it — non-zero exit on a bad rule
opsforge guard test "terraform destroy" --context prod --json  # assert in CI that prod destroys are denied
```

---

## Beyond the basics

### Stack profiles

A profile is just a named bundle of tools. Install a whole stack in one command,
or save your own:

```sh
opsforge install --profile aws-k8s   # aws, eksctl, kubectl, helm, k9s, terraform…
opsforge profiles                    # list all with install status
```

The built-in ones are `core`, `k8s`, `aws-k8s`, `gcp-k8s`, `iac`,
`observability`, `security`, `sysadmin`, `netsec`, `secrets` and `ai`. Want your
own? In the picker, select your tools and press `s` to save a personal profile to
`~/.config/opsforge/profiles.yaml`. From then on,
`opsforge install --profile my-stack` reproduces it anywhere.

### Workstation as code

Your machine setup shouldn't be a one-off snowflake you rebuild by hand every
time:

```sh
opsforge snapshot -o my-setup.yaml    # tools + profiles + shell + theme + guards + version manager → one file
opsforge apply <file-or-url>          # rebuild it on any machine
opsforge apply --check <file-or-url>  # verify a machine against it, without changing a thing
```

A snapshot captures the **whole** managed workstation: installed tools, your
custom profiles, the shell-environment state, the active **theme**, your **guard
policy** (the raw `guards.yaml`), and the detected **version manager**. When you
`apply` it, opsforge shows you the full plan and asks before changing anything
(`--yes` skips the prompt for scripts), then restores the theme and guard rules
right alongside the tools.

**Check a machine against a known-good snapshot.** `apply --check` compares this
machine to a snapshot **you froze earlier**, touching nothing, and exits
**non-zero on drift** (a missing tool, or a theme/guards/shell/version-manager
that no longer matches). Add `--json` and you get a structured report,
`{compliant, missing_tools, drift}`, so a CI job can assert that your laptop, or
a build image, still lines up with your reference setup:

```sh
opsforge apply --check my-setup.yaml            # fails the job on any drift
opsforge apply --check my-setup.yaml --json | jq '.compliant'
```

Snapshots are **forward-compatible**. The format grew from v1 (tools, profiles,
shell) to v2 (which adds theme, guards and version manager), and older v1 files
still load fine. The newer fields simply stay unset.

### Security audit

```sh
opsforge audit             # CVEs in your installed tools
opsforge audit --secrets   # + credentials leaking in history / rc / .env
```

It checks your installed versions against [OSV.dev](https://osv.dev) (the open
vulnerability database), sorts by severity, and tells you the version that fixes
each one:

```
⚠ argocd         2.11.0
    [CRITICAL] CVE-2025-47933 Argo CD allows cross-site scripting…  → fixed in 2.13.8
    [HIGH]     CVE-2025-59531 Unauthenticated argocd-server panic…  → fixed in 2.14.20
✓ helm           4.2.3 — no known vulnerabilities
```

The matching happens on your machine against OSV's affected ranges, so a CVE that
was fixed before your version (or only lands in a future major) won't nag you.
The `--secrets` flag scans your shell history, rc files and local `.env`s for the
usual leaks (AWS/GitHub/GitLab/Slack tokens, private keys, `--from-literal`,
`docker login -p` and the like) and always masks the values it finds.

### Pinning tool versions

```sh
opsforge install mise
opsforge use terraform@1.5   # pins it in this directory
```

This just hands off to **mise** (preferred) or **asdf**. There's no point
reinventing a version manager that already works.

---

## The DevOps shell environment

```sh
opsforge shell install && exec zsh    # zsh
opsforge shell install && exec fish   # fish (auto-detected from $SHELL)
opsforge shell install && exec bash   # bash
```

This takes your **own zsh, fish or bash** and makes it DevOps-aware. It doesn't
replace your shell, it enhances the one you already use. (`shell install`
auto-detects your shell from `$SHELL`, or you can force it with
`--shell zsh|fish|bash`; `shell uninstall` puts everything back the way it was.)

> **zsh · fish · bash.** The prod-aware prompt, the `?` inline help and the
> DevOps aliases work in all three. Two caveats are worth stating plainly:
> - The interactive niceties below (inline suggestions, syntax highlighting,
>   prefix history search) are **built into fish**, so opsforge just adds the
>   guards and prompt there. On **zsh** it installs the plugins that provide
>   them. **bash** has no standard equivalent, so it falls back to plain
>   readline.
> - The **blocking guard** is fully reliable on **zsh and fish** (both can
>   cancel a command before it runs). **bash can't** do that cleanly, so its
>   guard is *best-effort*: it confirms and warns, but backgrounding and some
>   multi-line constructs behave differently. If you want a hard block, use zsh
>   or fish. (Guards are a safety net, not a security boundary, in any shell.)
>
> The walkthrough below describes the zsh experience.

The details, then. opsforge turns your **own zsh** into a DevOps-aware
environment (its modules live under `~/.config/opsforge/shell/`, and `shell
uninstall` restores everything):

- **Calm, on-demand editing.** Nothing pops open in your face as you type, just a
  grey inline suggestion pulled from your history. `↑`/`↓` search history by the
  **whole-line prefix** you've typed, `→` accepts the whole suggestion, `Tab`
  takes it one word at a time, and the line is syntax-colored as you go. Even
  terraform (which ships no zsh completion of its own) and opsforge itself are
  covered.

  <table>
  <tr><th align="left">Key</th><th align="left">What it does</th></tr>
  <tr><td><code>↑</code> / <code>↓</code></td><td>Walk history by the line prefix you've typed (<code>kubectl get pods -n s</code> + <code>↑</code> cycles only lines starting that way)</td></tr>
  <tr><td><code>→</code></td><td>Accept the whole grey suggestion</td></tr>
  <tr><td><code>Tab</code></td><td>Accept the grey suggestion one word at a time (<code>ansible-play</code> + <code>Tab</code> → <code>ansible-playbook </code>)</td></tr>
  <tr><td><code>Ctrl-Space</code></td><td>File / command completion</td></tr>
  <tr><td><code>Ctrl-R</code></td><td>Search your whole history</td></tr>
  </table>

  Miss the old always-open live menu (zsh-autocomplete)? Set
  `OPSFORGE_AUTOMENU=1`. Or turn the whole layer off with
  `OPSFORGE_INTERACTIVE=0`.
- **`export <Tab>` knows the ecosystem.** Native zsh only completes variables
  that *already exist* in your session — so a var you've never set (say
  `AWS_SECRET_ACCESS_KEY`) is never offered and you retype it in full. opsforge
  teaches `export`/`typeset` the **standard variables of the tools it manages**
  (AWS, GCP, Azure, Kubernetes, Terraform, Vault, Docker, GitHub…), each with a
  one-line description. `export AWS_<Tab>` lists `AWS_ACCESS_KEY_ID`,
  `AWS_SECRET_ACCESS_KEY`, `AWS_REGION`, `AWS_PROFILE`… For the safe, enumerable
  ones it also completes the **value** — `AWS_PROFILE=<Tab>` reads your
  `~/.aws/config`, `AWS_REGION=<Tab>` lists regions, `TF_LOG=<Tab>` offers the
  log levels. It **never** completes a secret's value (keys, tokens, passwords):
  there's nothing safe to suggest and it won't read one from disk. Disable with
  `OPSFORGE_ENVCOMP=0`.
- **`?` help.** Press `?` on an empty line for a themed cheat-sheet. Type
  `kubectl get ?` and you get that command's help, rendered under a framed header
  with `bat`-colored man syntax. Type `??` to have an AI explain your last
  failure.
- **A prompt that knows where it is.** The right-hand prompt shows the kube
  `cluster:namespace` and turns **red the moment the context looks like prod**.
  That's a passive *visual* alarm you notice **before you even start typing**,
  next to the cloud account and terraform workspace (each shown only when it's
  relevant). The left prompt stays clean: repo-relative dir, git branch with
  dirty/ahead/behind markers, how long the last command took, and a `❯` that
  goes red on failure. Everything is read locally. No cloud or cluster is ever
  contacted. Prefer **starship**? Install it (`opsforge install starship`) and
  opsforge initializes it for you and steps aside — no `~/.zshrc` edit, no double
  prompt (`OPSFORGE_STARSHIP=0` keeps the opsforge prompt instead).
- **Policy-as-code guards.** Before a destructive command (`kubectl delete`,
  `terraform destroy`, `helm uninstall`…) on a production context, opsforge can
  confirm, warn, or block. It's driven by [rules you write in a
  file](#policy-as-code-guards), not by checks hard-coded into the tool. Set
  `OPSFORGE_GUARDS=0` to disable.
- **Aliases & helpers.** `k`, `tf`, `dc`, plus `kx`/`kn` to switch kube
  context and namespace (with an fzf picker when it's available). The `history`
  builtin is widened to show the last **200** lines (`history 1` for the lot),
  and `hg <term>` greps your whole history, while
  [`opsforge history`](#history) groups it by DevOps tool family.
- **A proactive heads-up.** Once per session, opsforge prints a compact one-liner
  in your own shell when something needs attention: a CVE just hit an installed
  tool, updates are waiting, a secret is leaking, or a newer opsforge is out. It
  reads a local cache (`~/.cache/opsforge/`, 6h TTL) and refreshes a stale one in
  the background, so your prompt never blocks on the network. Run
  [`opsforge notify`](#the-notify-digest) for the full breakdown, or silence the
  heads-up with `OPSFORGE_NOTIFY=0`.
- **Integrations.** `fzf`, `zoxide`, `atuin` and `starship` are wired up
  automatically when they're present.

**Three layers, three different jobs.** The **prompt** is a *passive* alarm: it
reddens so you **see** you're on prod. The [**guards**](#policy-as-code-guards)
are an *active* barrier: they **stop** a destructive command. And the
[**notify** heads-up](#the-notify-digest) is a *proactive* watch: it **tells**
you when a CVE, update or leak lands on your machine.

Every module is checked with `zsh -n` in CI, so a broken script can never make it
into your shell.

<table>
<tr><th align="left">Shell command</th><th align="left">What it does</th></tr>
<tr><td><code>opsforge shell install</code></td><td>Install the zsh environment into <code>~/.zshrc</code> (idempotent)</td></tr>
<tr><td><code>opsforge shell uninstall</code></td><td>Remove it cleanly (restores <code>~/.zshrc</code>)</td></tr>
<tr><td><code>opsforge shell doctor</code></td><td>Show what's provided and its state</td></tr>
<tr><td><code>opsforge shell sync</code></td><td>Refresh the shell modules <em>and</em> cached completions (run after upgrading opsforge)</td></tr>
</table>

### History

Your shell history is full of the exact commands you need again — buried under
everything else. `opsforge history` pulls out just one family of DevOps tools, so
you can find last week's `kubectl port-forward` without scrolling.

```sh
opsforge history              # overview: every family, with how many recent commands each has
opsforge history kube         # recent kubectl / helm / k9s / argocd… commands
opsforge history tf           # terraform / tofu / terragrunt
opsforge history git -n 50    # more results (0 = no cap)
opsforge history kube --json  # machine-readable
```

The built-in families group the tools you already think of together, and they
deliberately mirror the domains used by [guards](#policy-as-code-guards) and
profiles. So `kube`, `tf`, `cloud` and the rest mean the same thing everywhere in
the product:

<table>
<tr><th align="left">Family</th><th align="left">Tools</th></tr>
<tr><td><code>kube</code></td><td>kubectl, helm, k9s, kubectx, kustomize, stern, kubeseal, flux, argocd…</td></tr>
<tr><td><code>git</code></td><td>git, gh, glab, lazygit, tig</td></tr>
<tr><td><code>tf</code></td><td>terraform, tofu, terragrunt, tflint, terraform-docs</td></tr>
<tr><td><code>docker</code></td><td>docker, docker-compose, podman, nerdctl, colima</td></tr>
<tr><td><code>cloud</code></td><td>aws, gcloud, az, doctl, eksctl, flyctl, vercel</td></tr>
<tr><td><code>ansible</code></td><td>ansible, ansible-playbook, ansible-galaxy, ansible-vault</td></tr>
</table>

Pass a family name, or any executable to filter down to that single tool. You get
distinct commands, most-recent first, with a `×N` run count. `--limit/-n` caps
the list (default 20, `0` for all) and `--json` emits it for scripts. The history
is parsed **passively**: opsforge reads the file and never runs anything from
it.

---

## Encrypted env vault

Re-exporting the same `AWS_SECRET_ACCESS_KEY`, `VAULT_TOKEN` or registry
credentials in every new terminal is tedious — but putting them in `~/.zshrc`
persists them *in cleartext*, exactly the leak `opsforge audit --secrets`
(and any dotfiles backup) will find. `opsforge env` is the middle path: a
passphrase-locked vault that persists your variables **without ever writing a
secret in cleartext to disk**.

```sh
opsforge env set AWS_SECRET_ACCESS_KEY   # prompts for the value (masked) + a passphrase
opsforge env set AWS_DEFAULT_REGION=us-east-1   # non-secret? pass it inline
opsforge env list                        # variable NAMES only — never values
opsenv                                    # unlock into THIS shell (prompts once per session)
```

<table>
<tr><th align="left">Guarantee</th><th align="left">How</th></tr>
<tr><td>Nothing in cleartext at rest</td><td>The file <code>~/.config/opsforge/env.age</code> is encrypted with <a href="https://age-encryption.org">age</a> (scrypt passphrase). <code>cat</code>-ing it reveals nothing; <code>audit --secrets</code> ignores it.</td></tr>
<tr><td>Secrets never hit argv/history</td><td><code>env set NAME</code> reads the value <em>masked</em> from the terminal — it's never a command argument.</td></tr>
<tr><td>Loads into your real shell</td><td><code>opsenv</code> is a shell function (installed by the layer) that <code>eval</code>s the decrypted exports into the current session — the passphrase prompt goes to stderr, so it can't leak into the eval'd text.</td></tr>
</table>

**Honesty about the threat model:** once unlocked, the values are ordinary
environment variables in that session, visible to child processes. That's
inherent to using them (the AWS CLI must read the key). What the vault buys you
is: nothing in cleartext on disk, no retyping, and dotfiles you can back up or
commit without leaking. The crypto is [age](https://age-encryption.org) via its
reference Go library — opsforge rolls no crypto of its own.

---

## Policy-as-code guards

<div align="center">

![opsforge guard test — a prod terraform destroy denied by policy](demo/screenshots/guard.png)

</div>

Tools like Homebrew Bundle, mise, chezmoi and aqua install your CLIs. opsforge
adds a layer on top of that: it **guards how you use them**. The prod-safety
layer of the shell is really a small policy engine, a declarative set of rules
that decides whether a destructive command should run, warn, confirm, or be
refused, based on the context you're actually in.

### The two kinds of rule

Most guards fire when **two things line up at once**: a **destructive command**
*and* a **production marker**. Miss either one and the command runs untouched —
that's why read-only commands never nag you, and why destructive commands on
staging or dev stay out of your way.

The exception is a handful of **always-dangerous** commands that are irreversible
on *any* machine — `rm -rf /`, `dd … of=/dev/sdX`, `chmod -R 777`, `git clean -fdx`,
`curl … | bash`, `kubectl delete --all`. A production check would be pointless
there (`rm -rf ~/` on your laptop is just as unrecoverable), so those confirm
regardless of context — while staying tight enough that everyday variants like
`rm -rf ./build` still run untouched. It's a safety net for the distracted
gesture, not a wall in front of every command you type.

| Command | Context | Decision | Why |
|:--|:--|:--:|:--|
| `kubectl delete pod api` | `prod-eks` | ⚠️ confirm | destructive + prod |
| `kubectl get pods` | `prod-eks` | ✓ allow | prod, but read-only |
| `kubectl delete pod api` | `staging` | ✓ allow | destructive, but not prod |
| `terraform destroy -var-file=prod.tfvars` | *(none)* | ⚠️ confirm | prod is in the command itself |
| `terraform plan -var-file=prod.tfvars` | *(none)* | ✓ allow | plan is read-only |
| `helm uninstall app` | `prod` | ⚠️ confirm | destructive + prod |
| `rm -rf /var/lib/data` | *(none)* | ⚠️ confirm | always-dangerous — irreversible anywhere |
| `rm -rf ./build` | *(none)* | ✓ allow | a local build dir, not a broad path |
| `curl https://x.io \| bash` | *(none)* | ⚠️ warn | running an unverified remote script |
| `ls` · `git status` · `cat` | `prod` | ✓ allow | nothing destructive |

You can simulate any of these yourself with
`opsforge guard test "<cmd>" --context <ctx>`.

The built-in policy reaches well past Kubernetes and Terraform. It also catches a
**`git push --force` or `reset --hard` on `main`**, destructive **cloud** calls
(`aws s3 rm --recursive`, `ec2 terminate`, `eks/rds/cloudformation delete`,
`gcloud`/`az … delete` on prod), **container** footguns (`docker system prune`,
`volume rm`, `rm -f`) and **database** wipes (`FLUSHALL`, `DROP DATABASE` on
prod). In other words, the everyday commands worth a second look, not just the
obvious ones.

Rules live in a single file, `~/.config/opsforge/guards.yaml`. Each rule matches a
**command** regex against a **context** regex, and picks an action:

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

**Policy you can version and validate in CI.** Because the rules live in one file,
you can commit `guards.yaml` alongside your dotfiles and keep it honest in the
pipeline. Two commands make that possible:

- `opsforge guard lint` validates the active policy and **exits non-zero** when
  it's broken — a bad regex, unknown action, or wrong version fails the job
  instead of silently falling back to the default policy at runtime.
- `opsforge guard test "<cmd>" --context prod --json` emits the decision as
  `{command, context, matched_rule, action, message}`, so a pipeline can
  **assert** that, say, `terraform destroy` is `deny`ed on prod — the same
  `Evaluate` call the shell uses, so the test can't diverge from real behavior.

The guards run in your own shell, and the policy behind them is **testable and
versionable** like the rest of your setup, rather than a per-machine snowflake you
tweak by hand.

### An audit trail: what did I run on prod this week?

Every time a guard fires (warn, confirm, or deny), opsforge writes it to a local,
append-only trail. `opsforge guard log` replays it, answering the one thing your
shell history can't: *what did I run against a production context, and did the
guard let it through?*

```sh
opsforge guard log               # everything the guard flagged, oldest first
opsforge guard log --prod        # only production-like contexts
opsforge guard log --since 7d    # the last week
opsforge guard log --action deny # only what was blocked
opsforge guard log --json        # machine-readable
```

Your shell history says *what* you typed. This says *what was dangerous, in
which context, and how it was handled*, timestamp included. It lives only in
`~/.local/state/opsforge/guard-audit.jsonl` (mode 0600), never leaves your
machine, and logs only guarded commands, not your whole history (that's what
[`opsforge history`](#history) is for). It's best-effort, so a logging hiccup
never blocks the shell.

### How opsforge knows you're on prod

The "context" a rule matches against comes from **two places**, and opsforge is
deliberately upfront about the trade-offs of each:

- **Read passively from your environment**, without running a single command.
  opsforge reads the kubeconfig `current-context`, `AWS_PROFILE`/`AWS_VAULT`
  (or `CLOUDSDK_ACTIVE_CONFIG_NAME`), and the terraform workspace
  (`.terraform/environment`). It **never runs `kubectl` or `gcloud`** to figure
  out where you are, so evaluating a rule can't trigger an OIDC browser login or
  hang on a wrapper CLI.
- **Read from the command itself.** In 2026, teams target prod far more often with
  `-var-file=prod.tfvars` or an `environments/prod/` directory than with a
  terraform *workspace*. So the default policy also looks for those markers **in
  the command line** for `terraform`/`tofu`/`terragrunt`. That way
  `terraform destroy -var-file=prod.tfvars` prompts for confirmation even with no
  workspace set, while `terraform plan …` stays allowed because it's read-only.

> **Be clear-eyed about what this is.** Guards are a **safety net against the
> distracted gesture**. They catch you when you switch env without noticing, not
> when you're determined to do something. They are **not** a security boundary.
> Real prod protection stays where it belongs: `prevent_destroy`, separate cloud
> accounts, and CI approvals. opsforge **complements** that layer, it doesn't
> replace it.

### What you see when a guard fires

On a `confirm`, the command is held at the prompt until you type `yes`:

```text
⚠  opsforge guard
   This changes PRODUCTION Kubernetes resources.
   kubectl delete pod api -n payments
   (to skip guards this session: OPSFORGE_GUARDS=0)
Type 'yes' to run this: ▏
```

A `deny` prints a red **✗ Blocked by opsforge guard** and clears the line. A
`warn` prints its message and runs anyway.

### Everything is configurable in one file

- **Zero-config by default.** With no `guards.yaml`, the built-in policy above
  reproduces the old prod-confirm behavior exactly, so upgrading changes nothing
  until you opt into custom rules. When you're ready to customize, run
  `opsforge guard init`, which drops a fully commented `guards.yaml` you can
  edit.
- **Fast on the hot path.** The shell pre-filters cheaply and only calls the
  engine (`opsforge guard check`, used internally) on commands that actually look
  destructive, so your prompt stays instant.
- **Fails open, loudly.** A broken `guards.yaml` can never wedge your shell. The
  command runs, and the parse error goes to stderr so you can fix your YAML.

You can disable everything for one session with `OPSFORGE_GUARDS=0`.

### Guards at team scale

opsforge is a personal tool by design: no server, no fleet console. But the two
pieces a team actually needs to standardize *are already just files*, which is the
whole point of policy-as-code:

- **One policy, shared like any code.** `guards.yaml` is a plain, versionable
  file. Commit it to a shared repo, ship it in your dotfiles or a golden image,
  and every engineer runs the same rules, reviewed in a PR and tested in CI with
  `opsforge guard lint` / `guard test --json`. Nobody hand-tweaks a snowflake.
- **One audit trail, exportable to where your team already looks.** The guard
  log is newline-delimited JSON at a known path
  (`~/.local/state/opsforge/guard-audit.jsonl`), and `opsforge guard log --json`
  emits the same. Tail it with Promtail/Vector/Fluent Bit into Loki or your SIEM,
  and "what did anyone (or any [AI agent](#ai-agents-mcp)) run past a prod guard
  this week" becomes a fleet-wide query instead of a per-laptop one.

The path from *my workstation* to *the team's* is deliberately a config-and-log
story, not a platform you have to run. You adopt the parts you need, and opsforge
never becomes a dependency sitting in the middle.

---

## Project mode

<div align="center">

![opsforge sync --check — a drift report for a project's opsforge.yaml](demo/screenshots/sync.png)

</div>

A workstation snapshot pins a whole *machine*. A **project** usually needs less
than that: just the toolchain *this repo* depends on. Commit an `opsforge.yaml`
at its root and anyone can reproduce it with one command. It's the same
reproducibility mise and devbox give you, with a CVE gate added on top.

```yaml
# opsforge.yaml — commit at the repo root
version: 1
tools:
  - kubectl
  - helm
  - terraform
profiles:
  - core          # pull in whole stack profiles too
fail_on: high     # optional: sync fails if a required tool has a HIGH/CRITICAL CVE
```

```sh
opsforge sync           # install whatever the manifest declares that's missing
opsforge sync --check   # report drift, exit non-zero if a required tool is missing (CI/pre-commit)
opsforge sync --init    # write a starter opsforge.yaml here
```

`sync` walks up from your working directory to the nearest `opsforge.yaml`, so it
works from any subdirectory. It resolves `tools` and `profiles` into one
de-duplicated list, installs only what's missing (via Homebrew or a GitHub
release, chosen per tool), and skips anything not in the catalog with a warning.

**A CVE gate in the same file.** Set `fail_on: high` (or `critical`) and `sync`
audits *just the tools this project requires* against
[OSV.dev](https://osv.dev), then **fails** when one carries a CVE at that level.
So a single committed file gives you both a **reproducible environment** *and* a
**supply-chain gate** in one place. With `--json`, `sync --check` emits
`{compliant, missing, present, unknown, cve_blocked, fail_on}` for a pipeline to
assert on:

```sh
opsforge sync --check --json | jq '.compliant'   # fails the job on drift or a blocked CVE
```

**A lockfile for verifiable reproducibility.** `opsforge sync` also writes an
**`opsforge.lock`** next to the manifest, pinning each installed tool to its
exact resolved version. It's the same idea as `package-lock.json` or `mise.lock`.
Commit it, and `sync --check` no longer just verifies a tool is *present*: it
verifies it's the *pinned version*, flagging **version drift** in both the human
and JSON output:

```yaml
# opsforge.lock — written by sync, checked by sync --check (commit it)
version: 1
tools:
  - name: helm
    version: 3.14.0
  - name: kubectl
    version: 1.29.3
```

```sh
opsforge sync --check --json | jq '.version_drift'
# [{"name":"helm","expected":"3.14.0","got":"3.15.1"}]  → non-zero exit
```

It's non-breaking. With no lockfile, `--check` behaves exactly as before, and a
tool pinned with an unknown version is never flagged. That's what turns
"workstation-as-code" from a wish into reproducibility a reviewer can trust:
`opsforge.yaml` declares *what*, and `opsforge.lock` proves *which exact
version*.

---

## SBOM & supply-chain

<div align="center">

![opsforge sbom --audit — a CycloneDX SBOM with an embedded CVE, piped through jq](demo/screenshots/sbom.png)

</div>

An SBOM is the list of ingredients for your machine: every tool, its version, and
where it came from. opsforge emits a **CVE-correlated SBOM of your workstation**,
a supply-chain artifact that grype, `trivy sbom` or a compliance pipeline can
read straight away.

```sh
opsforge sbom                # CycloneDX 1.6 JSON of your installed tools → stdout
opsforge sbom --audit > bom.json   # + embedded CVE findings, captured to a file
```

- **`opsforge sbom`** builds a **CycloneDX 1.6** document where each installed
  tool is a component with its detected **version** and, when the catalog maps it
  to a package ecosystem, a **PURL** (a package URL that names it unambiguously).
- **`opsforge sbom --audit`** cross-references OSV.dev and embeds the known CVEs
  as CycloneDX **vulnerabilities**, each linked to its component with a severity
  and the recommended fix version. The SBOM ships CVE-corrected out of the box.

The document goes to stdout and a short summary to stderr, so
`opsforge sbom > bom.json` gives you a clean file plus feedback on screen: a
signed inventory of your toolbox *with* its vulnerabilities, ready to feed a
scanner or a compliance gate.

That's the whole supply-chain story in one binary. A **checksum** proves each
download is intact, a **cosign signature** proves the release is authentic (see
[the catalog](#the-catalog)), and the **SBOM** proves what you actually ended up
with, CVEs and all.

### VEX & CISA KEV

A raw CVE list tells you a vulnerability *exists*. It doesn't tell you which of
the dozens to fix **first**, and since the NVD stopped enriching most CVEs in
2026, the CVSS score you'd normally sort by is often missing or stale. VEX (a
"vulnerability exploitability exchange" document) is the artifact that carries
that verdict, and `opsforge vex` produces it.

```sh
opsforge vex                 # OpenVEX document → stdout (pairs with `opsforge sbom`)
opsforge vex --kev           # + highlight the actively-exploited (CISA KEV) CVEs
opsforge vex > vex.json      # capture the machine artifact
```

- **`opsforge vex`** turns the audit into an **[OpenVEX](https://openvex.dev)
  v0.2.0** document: one machine-readable statement per (component, CVE) with a
  status (`affected`) and an **action**, either upgrade to the fixed version or
  monitor the advisory when none exists yet. Each component is identified by the
  **same PURL the SBOM uses**, so a downstream scanner or auditor correlates the
  two out of the box. The output is deterministically sorted, so it diffs and
  signs cleanly.
- **`opsforge vex --kev`** cross-references **CISA's Known Exploited
  Vulnerabilities** catalog (the CVEs attackers are actually using) and calls out
  the ones being **exploited in the wild**: the handful to fix *now*, ahead of
  the long tail. The catalog is fetched once and cached
  (`~/.cache/opsforge/kev.json`, 24h TTL). It's best-effort, so a network hiccup
  degrades to "no KEV data" rather than a failed command.

Prioritizing by **exploitability** instead of by a score that may not even exist
is a sensible way to triage in 2026, and VEX is what carries that verdict to
whatever consumes it next.

### Signing the artifacts

Both the SBOM and the VEX document can be signed into a self-contained
**[Sigstore](https://www.sigstore.dev) bundle** you can hand to anyone (Sigstore
is the standard tooling for signing and verifying software artifacts):

```sh
opsforge sbom --sign > bom.json      # + a bom.sigstore.json bundle
opsforge vex  --sign > vex.json      # + a vex.sigstore.json bundle
cosign verify-blob --key ~/.config/opsforge/signing.pub \
  --bundle bom.sigstore.json bom.json
```

`--sign` signs the document **key-based** with a persistent local opsforge key
(an ECDSA P-256 key generated the first time you use it, under
`~/.config/opsforge/`) and writes a Sigstore bundle that `cosign verify-blob`, or
any Sigstore verifier, accepts. It's fully **offline**: no OIDC login, no
certificate, no entry in a public transparency log.

That last part is deliberate, and worth spelling out:

- **Local signing is key-based, on purpose.** Keyless Sigstore signing would
  publish the signer's OIDC identity (your email) in Rekor, a public and
  immutable log, on *every* signature. And it would prove nothing about
  supply-chain *provenance* for a document you generated by hand on a laptop. So
  opsforge signs with a local key instead.
- **Be clear about what it proves.** A key-based signature proves the document's
  **integrity** and **attribution to your key**, not that it was built by a
  trusted pipeline. Provenance is a CI property, so opsforge's own *releases* are
  the ones signed **keyless with SLSA provenance** (see
  [the catalog](#the-catalog)), because there the identity *is* the pipeline.

Same primitives, the right tool for each job: local integrity for the artifacts
you generate, keyless provenance for the binaries you ship.

### Scanning a container image

<div align="center">

![opsforge scan node:16-alpine --diff — image CVEs plus workstation correlation](demo/screenshots/scan.gif)

</div>

`opsforge scan` points the same OSV engine at a container image, and adds the one
thing a standalone scanner can't: **correlation with your own workstation**.

```sh
opsforge scan node:16-alpine          # CVEs in the image
opsforge scan my-ci-image --diff      # + how it drifts from your machine
opsforge scan my-image --json         # machine-readable, non-zero on HIGH/CRITICAL
```

opsforge **doesn't re-implement image SBOM extraction**. That's syft and trivy's
job, and importing them as libraries would bloat the binary for no gain. Instead
it drives whichever one is installed (the same way it hands version pinning off to
mise/asdf), reads back the CycloneDX SBOM, and runs those components through
opsforge's **own** OSV engine, the exact matcher `opsforge audit` uses on your
machine, CVSS scoring and per-branch fix versions included.

Add **`--diff`** and it answers a question trivy doesn't: *does a tool I run
locally ship at a different version inside this image?* It lines the image's
components up against your installed toolbox and reports the version drift, the
workstation-vs-CI skew that "works on my machine" tends to hide. Like `audit`, it
exits non-zero on a HIGH/CRITICAL CVE, so it drops into a pipeline as a gate.

> Needs `syft` or `trivy` on PATH (`opsforge install syft`). opsforge brings the
> correlation and the shared OSV verdict, not yet another image scanner.

### The notify digest

opsforge doesn't wait for you to run `audit`. `opsforge notify` is **one digest
of everything on *your* machine that needs attention**, gathered in a single
place:

- installed tools carrying a **known CVE** (HIGH/CRITICAL called out in red),
- tools that **can be updated**,
- **credentials leaking** in your shell history / rc / `.env` (when scanned),
- a **newer opsforge** than the one you're running.

Each line comes with the exact command that fixes it:

```
  ✗ 1 tool with a HIGH/CRITICAL CVE          → opsforge audit
  ✗ 6 critical secrets leaking in your shell → opsforge audit --secrets
  ⚠ 3 tools can be updated                   → opsforge upgrade -u
```

```sh
opsforge notify            # the full digest, grouped by severity
opsforge notify --json     # the structured Digest, for scripts
opsforge notify --refresh  # recompute the cache now
opsforge notify --quiet    # just the compact one-liner (used by the shell)
```

**A heads-up in your shell, once per session.** When something needs your
attention, the [DevOps shell](#the-devops-shell-environment) prints a compact
one-liner on startup, e.g. *"opsforge: 1 tool with a HIGH/CRITICAL CVE · 3
tools can be updated — run `opsforge notify`"*, and then you run `opsforge notify`
for the breakdown. Silence it with `OPSFORGE_NOTIFY=0`.

**Cached, instant, never blocking.** `notify` reads a local cache under
`~/.cache/opsforge/` (6h TTL) and only ever *reads* it. A stale cache is
refreshed in the background (or on demand with `--refresh`), so neither the
digest nor the shell heads-up ever waits on the network. The same finding also
surfaces at a glance in [`opsforge status`](#a-quick-tour).

It folds CVEs, updates, leaked secrets *and* its own self-update into one digest
and surfaces it, proactively, in your shell. So the moment an advisory lands on
your toolbox, you know, without having to run a thing.

---

## Credential hygiene (`verify`)

`opsforge audit` asks *are my tools vulnerable?* `opsforge verify` asks the other
half: *are the credentials sitting on this machine a liability?* A DevOps
workstation piles up secrets over time (cloud keys, a kubeconfig, SSH keys,
registry tokens, a heap of `~/.docker`/`~/.npmrc`/`~/.netrc` logins). `verify`
inventories them and flags the hygiene risks, in one read-only pass:

- **Long-lived static keys** that never expire: an AWS `AKIA` access key, a GCP
  service-account JSON key, an SSH key with no passphrase, a legacy static kube
  token. It tells a static key apart from a federated one (SSO, OIDC, `exec`,
  assume-role) and only flags the static ones.
- **Secrets in clear text**: the git credential store, `~/.netrc`, base64
  `~/.docker`/`~/.npmrc` logins (base64 is *not* encryption), `gh`/`glab` tokens.
- **Over-broad file permissions**: a credential file readable by more than its
  owner, when it should be `0600`.
- **Expired or soon-to-expire** client certificates and tokens, read straight
  from the local PEM/JWT.

```sh
opsforge verify            # human-readable report, most-severe first
opsforge verify --json     # machine-readable, for scripts
opsforge verify --strict   # exit non-zero on ANY finding (CI gate)
```

> **Read-only, offline, and honest.** `verify` **never runs an external tool**.
> In particular it never calls `kubectl`, so inspecting an OIDC kubeconfig can't
> trigger a login. It **never prints a secret's value** (only *where* it lives
> and *why* it's risky), and it **never touches the network**. It reads an OIDC
> kubeconfig by parsing YAML, a certificate's expiry from its PEM, a token's from
> its JWT claim, all passively. It's a safety net, not a guarantee: some stores
> (OS keychains) can't be read, and no findings isn't proof of safety.

Without `--strict`, the exit code is non-zero only on HIGH/CRITICAL findings, so
it gates CI without failing on every minor heads-up. That's the same convention
as [`opsforge audit`](#ci--integrations).

---

## AI agents (MCP)

opsforge speaks the **[Model Context Protocol](https://modelcontextprotocol.io)**,
MCP for short, the standard way an AI agent talks to a tool. So an agent (Claude
Code, Cursor, any MCP client) can *ask about your workstation* through the same
data the CLI computes, with no scraping and no guessing.

```sh
claude mcp add opsforge -- opsforge mcp   # register the stdio server once
```

`opsforge mcp` runs a stdio MCP server exposing **five read-only tools**:

| Tool | What the agent gets |
|:--|:--|
| `list_installed_tools` | every installed tool, its version, category, and whether it's outdated |
| `audit_vulnerabilities` | the CVEs on those tools (top severity + fixed-in), straight from OSV.dev |
| `generate_sbom` | a CycloneDX 1.6 SBOM (optionally with embedded CVEs) |
| `workstation_status` | one-glance summary: installed/outdated counts, shell state, kube/cloud/tf context |
| `check_guard_policy` | evaluate a command against your guard policy — `allow`/`warn`/`confirm`/`deny` — *before* the agent suggests running it |

> **Read-only by design.** Every tool is derived from read-only sources.
> **Nothing over MCP installs, upgrades, or changes the machine.** That's a
> deliberate boundary: an agent can *inspect* your workstation and *reason*
> about it (what's outdated, what carries a CVE, whether a command would trip a
> prod guard), but the mutating actions stay behind the interactive CLI, where
> *you* confirm them. `check_guard_policy` never runs the command, and, like the
> shell, reading the context never invokes `kubectl`/`gcloud`.

This turns opsforge into a **grounded source of truth** an agent can lean on.
Instead of hallucinating your tool versions or guessing whether `terraform
destroy` is safe here, it just asks.

**And you keep the receipts.** Every command an agent runs past
`check_guard_policy` that trips a guard (warn/confirm/deny) is written to the
same [audit trail](#an-audit-trail-what-did-i-run-on-prod-this-week) as your
own, tagged `source: mcp`. So you can go back later and see exactly what your AI
agents proposed against production:

```sh
opsforge guard log --source mcp          # what agents ran past the guard
opsforge guard log --source mcp --prod   # …only on production-like contexts
```

The guard becomes a **safety net between your agents and prod**. The agent checks
its intent before acting, and you get a reviewable record of every dangerous
command it considered, not just the ones it went through with.

#### See it in 30 seconds

An agent connected over MCP proposes a destructive command. The guard evaluates
it against the *current* context and refuses, **without ever executing it**, and
the attempt lands in your audit trail:

```console
# The agent calls the check_guard_policy MCP tool instead of running the command:
  → check_guard_policy(command="kubectl delete namespace payments",
                       context="gke_prod-eu")
  ← { "action": "confirm",
      "matched_rule": "confirm destructive kubectl on prod",
      "message": "This changes Kubernetes resources on a production-like context." }
  # The agent sees "confirm", stops, and asks you first. Nothing was deleted.

$ opsforge guard log --source mcp --prod
  2026-07-25 17:01  confirm  kubectl delete namespace payments
           context: gke_prod-eu  ·  via AI agent (MCP)
```

Same policy, same log, whether the risky command came from your fingers or your
agent. That's the whole differentiator in one screen.

---

## CI & integrations

opsforge isn't just a pretty TUI. A global `--json` flag makes `list`, `status`,
`doctor` and `audit` print structured JSON, so the same binary you use
interactively also drives scripts and pipelines.

```sh
opsforge audit --json | jq '.tools[] | select(.vulnerable)'   # only the affected tools
opsforge doctor --json | jq '.status'                         # "healthy" | "warnings" | "failing"
opsforge list all --json | jq '.[] | select(.outdated).name'  # tools with an update
```

The security commands also set **meaningful exit codes**, and that's what turns
opsforge into a one-line gate:

- `opsforge audit` (and `--json`) exits **non-zero on any HIGH/CRITICAL CVE**.
- `opsforge audit --secrets` adds leaked credentials to the report, and a
  **critical leak** exits non-zero too.
- `opsforge doctor --json` returns `{status, passed, warnings, failed, checks[]}`
  and fails when a check fails.

There's a ready-to-use GitHub Actions workflow at
[`examples/ci-security-gate.yml`](examples/ci-security-gate.yml). It installs
opsforge, fails the pipeline on any HIGH/CRITICAL CVE or leaked credential, and
uploads the JSON reports as artifacts.

```yaml
# excerpt — audit exits non-zero on HIGH/CRITICAL, failing the job on its own
- name: CVE audit
  run: opsforge audit --json | tee cve-report.json
```

### Official GitHub Action

Skip the install boilerplate. The composite action handles it, then runs
whichever gates you switch on (`audit`, `secrets`, `guard-lint`, `sbom`,
`baseline`):

```yaml
- uses: Mrg77/opsforge@v1
  with:
    audit: 'true'          # fail on any HIGH/CRITICAL CVE
    secrets: 'true'        # also fail on a leaked credential
    guard-lint: 'true'     # validate guards.yaml (policy-as-code)
    sbom: 'true'           # emit a CycloneDX SBOM, uploaded as an artifact
    vex: 'true'            # emit an OpenVEX doc (KEV-prioritized), uploaded too
    baseline: my-setup.yaml   # assert this machine matches your snapshot
```

Full example: [`examples/github-action-usage.yml`](examples/github-action-usage.yml).

### Docker image

A distroless image (~20–30 MB, no package manager) ships the static binary, so
you can run any command against a build image that already has your CLIs:

```sh
docker run --rm ghcr.io/mrg77/opsforge audit --json
```

This is the production image: minimal and non-interactive. If you want a
*playground* with a shell and the guards wired up instead, see
[Try it in a sandbox](#try-it-in-a-sandbox) (`ghcr.io/mrg77/opsforge-demo`).

### pre-commit hooks

Gate your commits with the same policy engine, straight from
[`.pre-commit-hooks.yaml`](.pre-commit-hooks.yaml):

```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/Mrg77/opsforge
    rev: v1.0.0
    hooks:
      - id: opsforge-guard-lint   # validate guards.yaml — fails on a bad rule
      - id: opsforge-secrets      # block a commit leaking a credential
```

---

## The catalog

**288 tools across 16 categories**: Kubernetes, Infrastructure as Code, Cloud
CLIs, Containers, Git & CI/CD, Observability & Monitoring, Logs, Networking &
HTTP, **System & SysAdmin**, Databases, Security & Compliance, Secrets & Identity,
Serverless & PaaS, Runtime & Versions, Utilities, and a new **AI & LLM** category.
The catalog now spans **every IT job**, not just Kubernetes and cloud but
development, DevOps, systems, networking, security, databases and AI. So a dev, a
DevOps engineer, a sysadmin, a network engineer, a DevSecOps or an AI engineer all
find their toolbox here:

- **Networking** — `tcpdump`, `iperf3`, `nmap`, `wireguard`…
- **System & SysAdmin** — `htop`, `tmux`, `zellij`, `rclone`…
- **Security & pentest** — `nuclei`, `ffuf`, `semgrep`, `trivy`, `opa`…
- **Databases** — `mongosh`, `litecli`, `atlas`…
- **Observability, GitOps & pipelines** — `prometheus`, `otel-cli`, `grafana`,
  `argo`, `tekton`/`tkn`, `dagger`…
- **AI & LLM** — `ollama`, `llm`, `aichat`, `mods`, `aider`, `fabric`,
  `gemini-cli`, `promptfoo`, `codex`…

The whole thing is a single embedded [YAML file](internal/catalog/catalog.yaml),
so adding a tool is a five-line PR.

**Two install backends, picked per tool at runtime:**

- **Homebrew** (when it's on PATH), always the latest release. `opsforge upgrade`
  refreshes the whole toolbox.
- **GitHub releases**, for hosts without Homebrew (bare Linux, CI images). Tools
  with a `github:` block are installed by downloading their release binary into
  `~/.local/bin`. No package manager required.

Force one with `OPSFORGE_BACKEND=brew|github`, and set the target dir with
`OPSFORGE_BIN_DIR`.

**Supply-chain: checksum verification.** Before a GitHub-release binary is made
executable, opsforge checks its **SHA-256 against a published checksum**, whether
that's `checksums.txt`, `<asset>.sha256`, or a template declared per tool via the
catalog's `checksum:` field. A mismatch **refuses the install**. A release that
publishes no checksum is a warning, not a failure (it's best-effort, so the ~85%
of projects that ship none still install).

**Supply-chain: signed provenance.** opsforge's own releases are **signed
keyless with [cosign](https://github.com/sigstore/cosign) (Sigstore)**. There's
no long-lived key; the certificate is bound to the release workflow's GitHub OIDC
identity, plus a native GitHub **SLSA build-provenance attestation**. The release
publishes `checksums.txt.sig` and `checksums.txt.pem` alongside `checksums.txt`.
On **self-update**, if `cosign` is installed locally, opsforge verifies that
signature against the expected identity and prints *"signature verified (cosign,
keyless)"*. A valid checksum whose signature does **not** verify is refused like a
mismatch. Want to check it yourself?

```sh
cosign verify-blob \
  --certificate checksums.txt.pem \
  --signature   checksums.txt.sig \
  --certificate-identity-regexp '^https://github.com/Mrg77/opsforge/\.github/workflows/release\.yml@.*' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  checksums.txt
```

### Add your own tools

The catalog isn't a closed list. Point opsforge at an **overlay** and your own
tools (internal or private CLIs) show up in the picker, in profiles and in every
command, **no PR required**. There are two ways to load one:

- Drop one or more files in `~/.config/opsforge/catalog.d/*.yaml` (merged in
  alphabetical order).
- Or set `OPSFORGE_CATALOG=/path/to/my-catalog.yaml`.

The format is exactly the catalog's own: `categories:` with `tools:` (`name`,
`bin`, `brew`, `description`), plus optional `profiles:`:

```yaml
# ~/.config/opsforge/catalog.d/internal.yaml
categories:
  - name: Internal
    tools:
      - name: acme-cli
        bin: acme
        brew: acmecorp/tap/acme-cli
        description: ACME Corp internal deploy CLI
```

The merge rules are predictable:

- A tool with a **new name** is **added** to the catalog.
- A tool with an **existing name** **overrides** the built-in one, so you can pin
  an internal formula, swap a source, or tweak a description.
- A profile with an existing name is **replaced** the same way.
- **Unknown YAML fields are rejected**, so a typo fails loudly instead of being
  silently ignored.

That's how you fold your own internal or private CLIs into opsforge: keep an
overlay next to your dotfiles, and your in-house tooling installs the same way as
the public catalog.

---

## Themes

The whole UI is themeable — one palette drives every command:

```sh
opsforge theme              # list all themes with a color preview
opsforge theme dracula      # preview one
opsforge theme set dracula  # persist it — every command follows, no reload
```

The themes are `forge` (the default), `nord`, `dracula`, `gruvbox`, `light`,
`mono` and `auto`. `auto` matches your terminal background, and `mono` is
color-free for logs and CI. Whichever you pick drives **every command *and* the
interactive picker**: one palette, no stray default colors anywhere. Precedence
runs `$OPSFORGE_THEME` › saved (`theme set`) › auto.

---

## Language (English / French)

opsforge speaks **English and French**. The showcase commands — `status`,
`doctor`, `audit`, `advise`, `ai` — are fully localized, and the rest falls back
to clear English (so a missing translation is never a raw key).

```sh
opsforge status            # auto-detected from your locale
opsforge --lang fr status  # force French for one command
OPSFORGE_LANG=fr opsforge doctor
```

The language is picked once, from `--lang` › `$OPSFORGE_LANG` › your `$LANG` /
`$LC_ALL`, defaulting to English. `opsforge advise` even asks your AI backend to
reply in that language, so the whole session stays in one tongue. Adding a
language is adding one map in `internal/i18n` — English stays the source of truth.

---

## Engineering highlights

If you're reviewing the code, these are the parts worth a closer look:

- **Policy engine for the shell.** Prod guards aren't hard-coded `if`s. They're a
  declarative policy (regex × context → allow/warn/confirm/deny), first-match-wins,
  validated on load, with a behavior-preserving built-in default. Context is read
  passively (kubeconfig / env / tf workspace) so evaluation never triggers an OIDC
  login, and the shell only calls the engine on commands that look destructive.
- **One policy, three shells.** The guard and prompt logic lives in Go and is
  exposed as plain text commands (`guard check`, `guard prefilter`), so porting
  from zsh to **fish** and **bash** came down to the hook, not the logic: the
  zsh `accept-line` ZLE widget maps to fish's `bind enter` + `commandline -f
  execute`, and to bash's `bind -x` on Enter. A small `Shell` abstraction
  (`internal/shellcfg/shell.go`) parameterizes install/env/modules per shell, and
  every module is parse-checked in CI (`zsh -n`, `fish --no-execute`, `bash -n`).
  The write-up is honest about the limit: zsh and fish can cancel a command
  before it runs, but **bash can't** cleanly, so its blocking guard is
  best-effort. That's a real trade-off, documented rather than hidden.
- **CVE audit with real version matching.** It queries OSV.dev per tool, filters
  vulnerabilities *client-side* against OSV's affected ranges (semver
  `introduced`/`fixed`) and dedupes CVEs listed under multiple advisory IDs, so it
  reports only what affects the version you actually run, with the fix on your
  branch. Severity comes from a real **CVSS v3.1 base-score computation** off the
  OSV vector, so a critical CVE is never mis-ranked or missed.
- **Supply-chain checksum verification.** GitHub-release binaries are SHA-256
  checked against a published checksum (`checksums.txt`, `<asset>.sha256`, or a
  catalog `checksum:` template) *before* they're made executable. A mismatch
  refuses the install, and a release with no checksum degrades to a warning.
- **A self-update that verifies its own integrity, and its provenance.**
  `opsforge self update` fetches the latest release, checks its published
  SHA-256, and only then replaces the running binary, atomically. The same
  supply-chain guarantee the installer gives your tools, opsforge applies to
  itself: a tampered asset is never made executable. Because our releases are
  **cosign-signed keyless**, self-update also **verifies that signature** (when
  cosign is installed) against the release-workflow OIDC identity, and a
  published-but-invalid signature is refused like a mismatch. `--check` reports
  availability with an exit code for cron/CI, and a dev build (no release tag to
  compare against) is a safe no-op.
- **Keyless-signed releases with SLSA provenance.** Releases are signed with
  **cosign keyless (Sigstore/Fulcio)** off the GitHub Actions OIDC identity, so
  there's no key to store, and they carry a native GitHub **SLSA build-provenance
  attestation**. `checksums.txt.sig` and `checksums.txt.pem` ship on every
  release, and anyone can `cosign verify-blob` them against the workflow identity.
- **One source of truth for tool families.** The DevOps "families" (`kube`,
  `tf`, `cloud`…) that `history` filters by and that the guard prefilter derives
  from now live in a single package (`internal/families`), the taxonomy that was
  once hard-coded in three diverging places. Adding a tool to a family, or a new
  danger verb, is a one-line change consumed everywhere at once.
- **Machine-readable, with exit codes that mean something.** A global `--json`
  flag renders `list`/`status`/`doctor`/`audit` as structured JSON, and `audit`
  exits non-zero on HIGH/CRITICAL CVEs (plus critical secret leaks with
  `--secrets`), so it drops into CI as a security gate with no wrapper script.
- **A CVE-correlated SBOM of your workstation.** `opsforge sbom` builds a
  CycloneDX 1.6 document from the *detected* tools (each a component with its
  version and, when mapped, a PURL), and `--audit` embeds the OSV.dev CVEs as
  linked CycloneDX vulnerabilities: a signed inventory of your toolbox *with* its
  vulnerabilities, feedable to grype/trivy or a compliance gate.
- **OpenVEX + exploitability triage.** `opsforge vex` re-uses the audit to emit
  an OpenVEX v0.2.0 document (one `affected` statement per (PURL, CVE) with an
  action), sharing the *exact* PURL the SBOM uses so the two correlate. `--kev`
  cross-references CISA's Known-Exploited catalog (cached, 24h TTL, best-effort)
  to surface what's exploited *in the wild*, a sensible way to prioritize now
  that CVSS enrichment is unreliable. The builder is pure (id/timestamp injected)
  and deterministically sorted, so the document diffs and signs.
- **Key-based Sigstore signing, chosen deliberately.** `sbom --sign` / `vex
  --sign` produce a self-contained Sigstore bundle via `sigstore-go` (a light
  dep, not cosign-as-a-library, which would bloat go.mod), implementing the
  `Keypair` interface over a persistent local ECDSA P-256 key so signing is
  fully offline and the public key stays stable across signatures. It's key-based
  rather than keyless on purpose: keyless would publish the signer's identity in
  a public Rekor log and prove nothing about provenance for a hand-generated
  document. So local signing proves integrity plus key attribution, and keyless
  provenance stays on the CI-signed releases. It's verifiable with `cosign
  verify-blob`, and the bytes signed are exactly the bytes written, so
  verification matches the file.
- **A read-only MCP server.** `opsforge mcp` exposes the workstation to AI agents
  over the Model Context Protocol via five tools (installed tools, CVE audit,
  SBOM, status, guard-policy check). The payload builders are pure functions over
  data opsforge already computes, unit-tested without a live client. Every tool
  is `ReadOnlyHint` and derived from read-only sources: the mutating commands stay
  behind the interactive CLI by design, so an agent can inspect but never change
  the machine.
- **A lockfile for verifiable reproducibility.** `opsforge sync` writes an
  `opsforge.lock` pinning each tool's exact resolved version (normalized,
  name-sorted for clean diffs). `sync --check` compares the machine against it
  and reports *version* drift, not just missing tools, in both JSON and human
  output, non-zero on mismatch. `opsforge.yaml` declares *what*, `opsforge.lock`
  proves *which version*, and it degrades gracefully (no lock means old behavior).
- **Image scanning by correlation, not reinvention.** `opsforge scan` drives an
  installed syft/trivy for the image SBOM (importing them as libraries would
  triple go.mod), then runs the components through opsforge's *own* OSV matcher
  and, with `--diff`, correlates them against the workstation toolbox to surface
  version drift a standalone scanner can't see. The reusable pieces
  (`internal/imagescan`: a purl→OSV parser, the correlation) are unit-tested, and
  the external SBOM step is delegated, on purpose.
- **OSV batch transport.** The audit finds every affected tool in one
  `/v1/querybatch` call, then fetches each distinct CVE once: fewer requests on
  the healthy path and OSV's rate-limit-friendly endpoint, with a per-tool
  fallback if the batch is down. The CVSS/semver matching engine is unchanged.
- **One cached digest, never blocking.** `opsforge notify` aggregates CVEs,
  available updates, leaked secrets and a newer opsforge into a single cached
  digest (`internal/notices`, `~/.cache/opsforge/`, 6h TTL). Both the shell (a
  once-per-session one-liner via `notify.zsh`) and `opsforge status` read it
  *without* a synchronous network call (a stale cache is recomputed in a detached
  background process), so the heads-up path can never hang your prompt. A fresh
  CVE, update or leak surfaces in your shell without you asking for it.
- **Reproducible env + a CVE gate in one file.** A committed `opsforge.yaml`
  (`version`, `tools`, `profiles`, `fail_on`) makes `opsforge sync` reproduce a
  repo's toolchain, and `fail_on: high|critical` audits *only the required tools*
  and fails the sync on a matching CVE. That's the reproducibility mise and devbox
  give you, plus a supply-chain gate in the same file.
- **Auth-safe detection.** Probing `kubectl --version` where kubectl is a
  cloud-SDK dispatcher wired to an OIDC plugin can pop a browser login. Every
  probe runs with a neutralized `KUBECONFIG` and a `WaitDelay`, so detection
  never triggers auth or hangs on a wrapper CLI holding the output pipe.
- **The catalog can't lie.** A CI job validates all 288 brew references against
  the Homebrew API and every GitHub asset template against the tool's real latest
  release (darwin/linux × amd64/arm64), so a renamed formula is caught before a
  user hits it mid-install.
- **Homebrew edge cases handled.** Auto-taps third-party taps and recovers from
  link conflicts (`brew link --overwrite`) that otherwise fail a docker upgrade.
- **Never breaks your shell.** Modules are `zsh -n`-checked in CI, and the `shell
  env` snippet does only PATH lookups (no subprocesses) for fast startup.

### Architecture

```
cmd/                Cobra commands (install, status, audit, guard, sync, sbom, vex, scan, mcp, snapshot, apply, self, doctor, shell, theme…)
internal/catalog/   Embedded YAML catalog + brew/github validation + OSV ecosystem mappings
internal/project/   opsforge.yaml manifest: resolve tools/profiles, drift plan, CVE gate (sync) + opsforge.lock (lock.go)
internal/sbom/      CycloneDX 1.6 builder (components + PURLs + embedded CVE vulnerabilities)
internal/vex/       OpenVEX v0.2.0 builder + CISA KEV fetch/cache (kev.go)
internal/attest/    Key-based Sigstore signing of the SBOM/VEX (local ECDSA key → Sigstore bundle)
internal/imagescan/ Container-image scan: syft/trivy SBOM → opsforge's OSV engine → workstation correlation
internal/mcp/        Read-only MCP payload builders (pure functions over catalog/detect/audit/guard)
internal/detect/    Concurrent PATH + version detection + brew-outdated
internal/installer/ Backend router: Homebrew + GitHub-releases download (checksum.go: SHA-256 verify; self-update)
internal/audit/     OSV.dev client + client-side version matching + CVSS v3.1 scoring
internal/credscan/  Read-only credential-hygiene scanner (static keys, clear-text secrets, perms, cert/JWT expiry) — never runs an external tool
internal/families/  Single source of truth for DevOps tool families (consumed by history + guard prefilter)
internal/history/   Passive shell-history reader + DevOps tool-family grouping
internal/secrets/   Leaked-credential scanner
internal/notices/   Cached digest behind `opsforge notify` (CVEs + updates + secrets + self-update)
internal/output/    Machine-readable JSON emitter for the --json flag
internal/snapshot/  Workstation capture / apply / --check drift report
internal/tui/       Bubble Tea picker with tabs (theme-bound styling)
internal/shellcfg/  zsh + fish + bash environment modules (modules/, modules/fish/, modules/bash/) + per-shell install (shell.go) + guard policy engine (policy.go)
internal/guardlog/  Append-only local audit trail of guard decisions (`opsforge guard log`)
internal/ui/        Shared visual identity + themes
```

---

## Development

```sh
go test ./...                                   # unit tests
OPSFORGE_SKIP_BREW_VALIDATION=1 go test ./...   # skip the network catalog checks
go build -o opsforge .
```

CI runs gofmt, vet and race tests on Linux and macOS, validates the catalog
against upstream, and cross-compiles every target. Releases are cut by GoReleaser
on tag.

## Roadmap

**Recently shipped**

- [x] `opsforge verify`, a read-only [credential-hygiene](#credential-hygiene-verify) audit of the workstation
- [x] `opsforge scan <image>`, an image CVE scan correlated with your workstation
- [x] `opsforge sbom/vex --sign`, key-based Sigstore signing of the artifacts
- [x] One-command interactive [demo sandbox](#try-it-in-a-sandbox) (Docker + Codespaces)
- [x] Read-only [MCP server](#ai-agents-mcp) for AI agents
- [x] `opsforge.lock` for verifiable, reproducible toolchains
- [x] **fish** support for the shell layer (guards, prompt, `?` help, aliases)
- [x] **bash** support for the shell layer (prompt, `?` help, aliases; the guard
      is best-effort, since bash can't cancel a command pre-execution like
      zsh/fish)

**Next**

- [ ] Native Windows (winget/scoop + PowerShell completions)
- [ ] More `github:` templates for full brew-less coverage

## License

MIT
