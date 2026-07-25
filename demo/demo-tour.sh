#!/usr/bin/env bash
# Guided tour for the opsforge demo sandbox.
#
# Runs a handful of READ-ONLY opsforge commands to show what the tool does,
# then drops you into the forged zsh shell (already in a fake "prod" context)
# so you can try a guarded command yourself. Nothing here touches real infra:
# the cluster is a one-line fake kubeconfig and kubectl/terraform/helm are
# no-op stubs on PATH.
set -euo pipefail

# Colors (fall back to nothing if the terminal is dumb).
if [ -t 1 ]; then
  B=$'\e[1m'; DIM=$'\e[2m'; CYAN=$'\e[36m'; GREEN=$'\e[32m'; R=$'\e[0m'
else
  B=""; DIM=""; CYAN=""; GREEN=""; R=""
fi

say()  { printf '\n%s%s%s\n' "$CYAN$B" "$1" "$R"; }
note() { printf '%s%s%s\n' "$DIM" "$1" "$R"; }
run()  { printf '\n%s❯ %s%s\n' "$GREEN$B" "$1" "$R"; eval "$1" || true; }
pause() { [ -t 0 ] && { printf '\n%s(press enter to continue)%s' "$DIM" "$R"; read -r _; } || sleep 1; }

clear || true
cat <<EOF
${B}opsforge — interactive demo sandbox${R}
${DIM}A throwaway container. You are in a FAKE prod context; kubectl/terraform/helm
are no-op stubs. Nothing you type here can reach real infrastructure.${R}
EOF
pause

say "1. Where am I? A one-glance look at this (sandbox) workstation."
run "opsforge status"
pause

say "2. The prod-safety policy — declarative guards, not hard-coded checks."
run "opsforge guard list"
pause

say "3. The differentiator: guards decide per (command × context)."
note "We're on context 'gke_…_prod'. Watch what the policy does with a"
note "destructive command here — evaluated safely, the command is NEVER run."
run "opsforge guard test 'kubectl delete namespace payments' --context prod"
run "opsforge guard test 'terraform destroy' --context prod"
note "…and the same commands away from prod stay out of your way:"
run "opsforge guard test 'kubectl delete namespace payments' --context staging"
pause

say "4. The audit trail: every guard decision is kept, so you can review it."
note "This sandbox already has a couple of prod entries from the build."
run "opsforge guard log --prod"
pause

say "5. Credential hygiene: is anything on this box a liability?"
note "This sandbox is seeded with FAKE credentials (an AWS key, a passphrase-less"
note "SSH key, a base64 docker login) so verify has something to flag. It is"
note "strictly read-only: it never runs a tool and never prints a secret's value."
run "opsforge verify"
pause

say "6. Supply chain: a CVE-correlated SBOM and an OpenVEX document of the box."
note "(offline in this sandbox — shown as JSON you'd pipe to a scanner)"
run "opsforge sbom --audit | head -c 600; echo"
pause

say "You're all set — dropping you into the forged zsh shell."
cat <<EOF
${DIM}Try it live (you are in fake prod):
  ${R}${B}kubectl delete namespace payments${R}${DIM}   → the guard intercepts it
  ${R}${B}opsforge verify${R}${DIM}                       → audit the (fake) credentials here
  ${R}${B}opsforge guard log --prod${R}${DIM}             → what tripped a guard on prod
  ${R}${B}opsforge --help${R}${DIM}                       → everything else
Type 'exit' to leave the sandbox.${R}
EOF
pause

exec zsh -l
