# opsforge demo — regenerate with: vhs demo.tape
Output demo-v0.3.2.gif

Set Shell "zsh"
Set FontSize 16
Set Width 1200
Set Height 760
Set Padding 24
Set Theme "Catppuccin Mocha"
Set TypingSpeed 65ms
Set PlaybackSpeed 1.0

Hide
# seed a realistic history so `opsforge history kube` has something to show
Type "printf '%s\n' 'kubectl get pods -n staging' 'kubectl logs -f deploy/api' 'helm upgrade api ./chart -n prod' 'kubectl rollout restart deploy/web' 'k9s -n observability' 'kubectl get pods -n staging' > /tmp/opsforge-demo-history" Enter
Type "export HISTFILE=/tmp/opsforge-demo-history" Enter
Type "fc -R /tmp/opsforge-demo-history" Enter
Type "clear" Enter
Show
Sleep 1s

# --- the cockpit -------------------------------------------------------
Type "opsforge status"  Sleep 500ms  Enter
Sleep 7s

Type "clear" Enter
Sleep 800ms

# --- the interactive picker (with tabs) --------------------------------
Type "opsforge"  Sleep 500ms  Enter
# let the full catalog view breathe — this is the richest screen
Sleep 6s
# scroll down a touch so the reader sees it is a long, browsable list
Down Down Down  Sleep 2s
Down Down Down  Sleep 2500ms
Up Up Up Up Up Up  Sleep 2s

# filter to k8s
Type "/"  Sleep 700ms
Type "k8s"  Sleep 3500ms
Escape  Sleep 2s

# tab 2: updates only
Type "2"  Sleep 4s
# tab 3: live CVE security scan
Type "3"  Sleep 6s
# back to tools
Type "1"  Sleep 2s
Type "q"  Sleep 1s

Type "clear" Enter
Sleep 800ms

# --- search the catalog: 249 tools, one term away ----------------------
Type "opsforge list dns"  Sleep 500ms  Enter
Sleep 5s

Type "clear" Enter
Sleep 800ms

# --- stack profiles ----------------------------------------------------
# 7 ready-made stacks; `opsforge install --profile <name>` installs one in one go
Type "opsforge profiles"  Sleep 500ms  Enter
# long screen: hold, then scroll through the stacks
Sleep 6s
Down Down Down Down  Sleep 3s
Down Down Down Down  Sleep 3s
Down Down Down Down  Sleep 3s

Type "clear" Enter
Sleep 800ms

# --- prod guards: policy-as-code for the shell (the differentiator) -----
# declarative rules block destructive commands on a prod context
Type "opsforge guard list"  Sleep 500ms  Enter
Sleep 5s

Type "clear" Enter
Sleep 700ms

# simulate: what happens if you run `terraform destroy` on prod?
Type "opsforge guard test 'terraform destroy' --context prod"  Sleep 500ms  Enter
Sleep 5s

Type "clear" Enter
Sleep 800ms

# --- sbom: a CVE-correlated CycloneDX SBOM of your workstation ---------
# the supply-chain differentiator — pipe it to jq to show it is real JSON
Type "opsforge sbom --audit | jq '{components: (.components|length), vulns: (.vulnerabilities|length)}'"  Sleep 500ms  Enter
Sleep 7s

Type "clear" Enter
Sleep 800ms

# --- history: recent commands, filtered by DevOps tool family ----------
Type "opsforge history kube"  Sleep 500ms  Enter
Sleep 5s

Type "clear" Enter
Sleep 800ms

# --- doctor: full health check incl. CVEs & leaked secrets -------------
Type "opsforge doctor"  Sleep 500ms  Enter
Sleep 8s
