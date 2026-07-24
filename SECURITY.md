# Security Policy

## Supported versions

Only the latest release receives security fixes. Update with
`opsforge self update` (checksum- and cosign-verified before the swap).

## Reporting a vulnerability

Please report suspected vulnerabilities **privately** via GitHub's
[private security advisory form](https://github.com/Mrg77/opsforge/security/advisories/new)
rather than a public issue.

I aim to acknowledge reports within a few days. As a solo-maintained personal
project there is no formal SLA, but security reports are prioritized over
feature work.

## Scope

opsforge installs third-party CLIs and runs shell guards locally, so it helps
to be clear about what belongs here:

- **In scope** — anything in opsforge's own code: the guard policy engine,
  SBOM/VEX generation, the CVE audit and its OSV matching, the checksum and
  cosign-signature verification on install and self-update, and the read-only
  MCP server.
- **Out of scope** — vulnerabilities in the *installed* tools themselves
  (report those upstream) and the guards' threat model: guards are a safety net
  against a distracted destructive command, **not** a security boundary, and
  are documented as such. A guard that can be bypassed is expected, not a
  vulnerability.

If you're unsure, report it anyway — I'd rather triage an out-of-scope report
than miss a real one.
