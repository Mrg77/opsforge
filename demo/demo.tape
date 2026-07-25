# opsforge demo — regenerate with: vhs demo.tape
# Requires on the host: opsforge + syft on PATH, docker running, and the demo
# image built for the sandboxed verify/guard-log scenes:
#   docker build -f demo/Dockerfile.demo -t opsforge-demo:demo .
#
# Narrative order leads with the MOAT (guards), then the security of the box
# (verify), then supply-chain (vex/sbom), and only THEN the installer — the
# commodity. `verify` runs INSIDE the throwaway demo container so no real
# credential from the host can ever appear on screen.
Output demo/demo.gif

Set Shell "zsh"
Set FontSize 16
Set Width 1200
Set Height 760
Set Padding 24
Set Theme "Catppuccin Mocha"
Set TypingSpeed 60ms
Set PlaybackSpeed 1.0

Hide
# Let the narration lines (typed as `# ...`) be treated as shell comments
# instead of commands, so they don't print "command not found: #".
Type "setopt interactive_comments" Enter
# `verify` and `guard log` run inside the throwaway demo container (seeded with
# FAKE credentials) so no real secret from the host can appear on screen. Alias
# it to `sandbox` so the command line stays readable.
Type "alias sandbox='docker run --rm --entrypoint opsforge opsforge-demo:demo'" Enter
# Warm the OSV/KEV caches so the vex/sbom steps render their result promptly
# on screen instead of waiting on the network mid-recording.
Type "opsforge vex --kev >/dev/null 2>&1; opsforge audit >/dev/null 2>&1" Enter
Sleep 12s
Type "clear" Enter
Show
Sleep 1s

# --- the cockpit: set the scene ----------------------------------------
Type "opsforge status"  Sleep 500ms  Enter
Sleep 5s
Type "clear" Enter
Sleep 700ms

# === THE MOAT: policy-as-code guards on destructive commands ===========
Type "# The differentiator: guards stop a destructive command on prod."  Enter
Sleep 1500ms
Type "opsforge guard test 'kubectl delete namespace payments' --context prod"  Sleep 500ms  Enter
Sleep 4s
# ...and every guard decision is kept in an auditable trail.
Type "sandbox guard log --prod"  Sleep 500ms  Enter
Sleep 5s
Type "clear" Enter
Sleep 700ms

# === SECURITY OF THE BOX: credential-hygiene audit (in the sandbox) =====
Type "# opsforge verify: a read-only audit of the credentials on the box."  Enter
Sleep 1500ms
Type "sandbox verify"  Sleep 500ms  Enter
Sleep 6s
Type "clear" Enter
Sleep 700ms

# === SUPPLY-CHAIN: triage CVEs by real-world exploitability ============
Type "opsforge vex --kev"  Sleep 500ms  Enter
Sleep 9s
Type "clear" Enter
Sleep 700ms

# ...and sign the SBOM into a Sigstore bundle.
Type "opsforge sbom --sign > bom.json"  Sleep 500ms  Enter
Sleep 7s
Type "ls bom.json bom.sigstore.json"  Sleep 400ms  Enter
Sleep 3s
Type "clear" Enter
Sleep 700ms

# === THE COMMODITY: it also installs & maintains the toolbox ===========
Type "# And because a safety layer is useless on an empty box, it installs too:"  Enter
Sleep 1500ms
Type "opsforge"  Sleep 500ms  Enter
Sleep 4s
Down Down Down  Sleep 1200ms
Type "/"  Sleep 500ms
Type "k8s"  Sleep 2000ms
Escape  Sleep 1200ms
Type "3"  Sleep 3s
Type "q"  Sleep 800ms
Type "clear" Enter
Sleep 500ms
