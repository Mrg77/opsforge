package i18n

// en is the English message table — the source of truth. Every key that any
// other language translates must exist here, so the fallback is always readable.
//
// Key convention: "<command>.<what>". Keep values free of leading/trailing
// spaces and of terminal styling; callers apply color. {name} marks a
// placeholder substituted via i18n.V.
var en = map[string]string{
	// ── status ────────────────────────────────────────────────────────────
	"status.short":              "A one-glance cockpit of your DevOps workstation",
	"status.header.tag":         "your DevOps workstation at a glance",
	"status.label.toolbox":      "Toolbox",
	"status.label.updates":      "Updates",
	"status.label.security":     "Security",
	"status.label.posture":      "Posture",
	"status.label.shell":        "Shell",
	"status.label.versions":     "Versions",
	"status.label.backend":      "Backend",
	"status.label.theme":        "Theme",
	"status.label.profiles":     "Profiles",
	"status.toolbox.installed":  "{n}/{total} installed",
	"status.updates.available":  "{n} available",
	"status.updates.hint":       "— run `opsforge upgrade -u`",
	"status.updates.uptodate":   "everything up to date",
	"status.security.pending":   "scan pending — `opsforge audit` for a full report",
	"status.security.highcrit":  "{n} tool(s) with HIGH/CRITICAL CVEs",
	"status.security.cves":      "{n} tool(s) with CVEs",
	"status.security.audithint": "— `opsforge audit`",
	"status.security.clean":     "no known CVEs",
	"status.posture.hint":       "— your workstation security posture · `opsforge advise` for a plan",
	"status.shell.off":          "off — `opsforge shell install`",
	"status.shell.active":       "active",
	"status.versions.none":      "none — install mise for `opsforge use`",
	"status.backend.github":     "GitHub releases",
	"status.backend.brew":       "Homebrew + GitHub",
	"status.theme.fromenv":      " (from $OPSFORGE_THEME)",
	"status.theme.auto":         " (auto — `opsforge theme set <name>` to change)",
	"status.footer":             "  Run `opsforge` to open the picker · `opsforge doctor` for a full check",
	"status.tip":                "  Tip: `opsforge audit --secrets` scans your tools for CVEs and your shell for leaked credentials",

	// ── doctor ────────────────────────────────────────────────────────────
	"doctor.short":      "Full health check of your DevOps workstation",
	"doctor.header.tag": "a full health check of your DevOps workstation",

	// section headers
	"doctor.section.system":   "System",
	"doctor.section.shell":    "Shell environment",
	"doctor.section.toolbox":  "Toolbox",
	"doctor.section.security": "Security",
	"doctor.section.health":   "Health",

	// System section
	"doctor.system.homebrew":           "Homebrew",
	"doctor.system.homebrew.available": "available",
	"doctor.system.homebrew.notfound":  "not found",
	"doctor.system.homebrew.fix":       "install from https://brew.sh (opsforge can also install via GitHub releases)",
	"doctor.system.brewpath":           "Homebrew bin on PATH",
	"doctor.system.brewpath.fix":       "add `eval \"$(/opt/homebrew/bin/brew shellenv)\"` to ~/.zprofile",
	"doctor.system.localbin":           "~/.local/bin on PATH",
	"doctor.system.localbin.detail":    "(GitHub-installed tools land here)",
	"doctor.system.localbin.fix":       "add `export PATH=\"$HOME/.local/bin:$PATH\"` to ~/.zshrc",
	"doctor.system.vm":                 "Version manager",
	"doctor.system.vm.works":           "{mgr} — `opsforge use <tool>@<ver>` works",
	"doctor.system.vm.none":            "not installed (optional — `opsforge install mise` enables `opsforge use`)",

	// Shell section
	"doctor.shell.layer":           "opsforge shell layer",
	"doctor.shell.active":          "active in ~/.zshrc",
	"doctor.shell.inactive":        "not installed",
	"doctor.shell.layer.fix":       "run `opsforge shell install`",
	"doctor.shell.completions":     "Cached completions",
	"doctor.shell.completions.n":   "{n} tool(s)",
	"doctor.shell.completions.fix": "run `opsforge shell sync`",
	"doctor.shell.plugin.fix":      "installed by `opsforge shell install`",

	// Toolbox section
	"doctor.toolbox.installed":        "Installed tools",
	"doctor.toolbox.installed.n":      "{n} of {total} catalog tools",
	"doctor.toolbox.updates.avail":    "Updates available",
	"doctor.toolbox.updates.detail":   "{n} tool(s): {list}",
	"doctor.toolbox.updates.fix":      "run `opsforge upgrade -u` to update them all",
	"doctor.toolbox.updates.ok":       "Updates",
	"doctor.toolbox.updates.uptodate": "everything up to date",
	"doctor.toolbox.probe":            "Version probe",
	"doctor.toolbox.probe.detail":     "{n} tool(s) report no version (cosmetic): {list}",

	// Security section
	"doctor.sec.cves":             "Known CVEs",
	"doctor.sec.cves.skipped":     "skipped (--skip-security)",
	"doctor.sec.cves.noauditable": "no auditable tool installed",
	"doctor.sec.cves.scanning":    "  scanning OSV.dev for CVEs…",
	"doctor.sec.cves.clean":       "{n} tool(s) checked, none vulnerable",
	"doctor.sec.cves.affected":    "{n} tool(s) affected: {list}",
	"doctor.sec.cves.fix":         "run `opsforge audit` for details, then `opsforge upgrade` the affected tools",
	"doctor.sec.secrets":          "Leaked secrets",
	"doctor.sec.secrets.clean":    "none in history, rc files or local .env",
	"doctor.sec.secrets.found":    "{n} potential leak(s) found",
	"doctor.sec.secrets.critical": " ({n} critical)",
	"doctor.sec.secrets.fix":      "run `opsforge audit --secrets`, then rotate and remove the exposed credentials",

	// Health summary
	"doctor.health.passed":   "{n} passed",
	"doctor.health.warnings": "{n} warnings",
	"doctor.health.failed":   "{n} failed",
	"doctor.health.failing":  "Some checks failed — address the → hints above.",
	"doctor.health.warned":   "Healthy, with a few optional improvements above.",
	"doctor.health.allgood":  "All good. Happy shipping! 🔥",
	"doctor.checks.failed":   "{n} check(s) failed",

	// flag help
	"doctor.flag.skipsecurity": "skip the online CVE scan (offline / faster)",

	// ── audit ──
	"audit.short": "Scan installed tools for CVEs — and your workstation for leaked secrets",
	"audit.long": `Cross-references the versions of your installed tools against the OSV.dev
vulnerability database and reports which ones have known CVEs and should be
upgraded. Only tools with an OSV mapping in the catalog are checked.

With --secrets, also scans the places credentials habitually leak — shell
history, shell rc files, and local .env files — and reports masked findings.`,
	"audit.header.sub.cves":       "installed tool versions vs the OSV.dev vulnerability database",
	"audit.header.sub.secrets":    "CVEs in your tools + credentials leaking on your workstation",
	"audit.noauditable":           "No auditable tools installed (no installed tool carries an OSV mapping).",
	"audit.scanning":              "Auditing {n} installed tool(s) against OSV.dev…",
	"audit.tool.clean":            "{version} — no known vulnerabilities",
	"audit.vuln.fixedin":          "  → fixed in {version}",
	"audit.allclean":              "All audited tools are free of known vulnerabilities.",
	"audit.found":                 "Found vulnerabilities",
	"audit.found.hint":            "in {n} tool(s). Run `opsforge upgrade` or update the affected tools.",
	"audit.secrets.scanning":      "Scanning your workstation for leaked secrets…",
	"audit.secrets.scanning.note": "  (shell history, shell rc files, local .env files — values are masked)",
	"audit.secrets.clean":         "✓ No leaked credentials found.",
	"audit.secrets.line":          "line",
	"audit.secrets.found":         "Found {n} potential leak(s)",
	"audit.secrets.found.hint":    "in {n} location(s).",
	"audit.secrets.cleanup": `  Clean up: rotate any real credentials, then remove the lines
  (history: edit ~/.zsh_history · prefer 'read -s' or a secrets manager next time)`,
	"audit.flag.secrets": "also scan shell history, rc files and local .env files for leaked credentials",

	// ── ai ────────────────────────────────────────────────────────────────
	"ai.short":        "Show which AI backend opsforge uses — or how to set one up",
	"ai.header.tag":   "the AI backend opsforge drives",
	"ai.none":         "No AI backend detected",
	"ai.backend":      "Backend: ",
	"ai.backend.note": "      opsforge feeds it a prompt and streams the answer — nothing is executed.",
	"ai.try":          "Try:",
	"ai.try.advise":   "opsforge advise    # AI-prioritized take on your CVEs, updates & secrets",
	"ai.override":     "Override the backend any time with $OPSFORGE_AI_CMD.",

	// ── advise ────────────────────────────────────────────────────────────
	"advise.short":      "Ask the AI to prioritize your workstation's CVEs, updates & secrets",
	"advise.header.tag": "AI-prioritized plan for your workstation",
	"advise.nobackend":  "no AI backend available",
	"advise.scanning":   "  Scanning (CVEs · updates · credentials)…",
	"advise.clean":      "Nothing flagged — your workstation looks clean.",
	"advise.asking":     "  Asking {backend}…",
	"advise.replylang":  "Reply in English.",
}
