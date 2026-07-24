# opsforge demo — regenerate with: vhs demo.tape
# Requires: opsforge + syft on PATH, docker with node:16-alpine pulled.
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
# Warm the OSV/KEV caches so the vex/sbom steps render their result promptly
# on screen instead of waiting on the network mid-recording.
Type "opsforge vex --kev >/dev/null 2>&1; opsforge audit >/dev/null 2>&1" Enter
Sleep 12s
Type "clear" Enter
Show
Sleep 1s

# --- the cockpit -------------------------------------------------------
Type "opsforge status"  Sleep 500ms  Enter
Sleep 6s
Type "clear" Enter
Sleep 700ms

# --- the interactive picker (tabs: Tools · Updates · Security) ----------
Type "opsforge"  Sleep 500ms  Enter
Sleep 5s
Down Down Down  Sleep 1500ms
Up Up Up  Sleep 1s
Type "/"  Sleep 600ms
Type "k8s"  Sleep 2500ms
Escape  Sleep 1500ms
Type "3"  Sleep 4s
Type "1"  Sleep 1500ms
Type "q"  Sleep 800ms
Type "clear" Enter
Sleep 700ms

# --- prod guards: policy-as-code for the shell (the differentiator) -----
Type "opsforge guard test 'kubectl delete namespace payments' --context prod"  Sleep 500ms  Enter
Sleep 4s
Type "clear" Enter
Sleep 700ms

# --- VEX + CISA KEV: triage CVEs by exploitability ---------------------
Type "opsforge vex --kev"  Sleep 500ms  Enter
Sleep 9s
Type "clear" Enter
Sleep 700ms

# --- sign the SBOM into a Sigstore bundle ------------------------------
Type "opsforge sbom --sign > bom.json"  Sleep 500ms  Enter
Sleep 7s
Type "ls bom.json bom.sigstore.json"  Sleep 400ms  Enter
Sleep 3s
Type "clear" Enter
Sleep 700ms

# --- doctor: full health check incl. CVEs & leaked secrets -------------
Type "opsforge doctor"  Sleep 500ms  Enter
Sleep 7s
