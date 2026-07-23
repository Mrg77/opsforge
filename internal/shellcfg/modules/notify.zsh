# opsforge notifications — a one-time, ambient heads-up on shell start when
# something on your machine needs attention: a CVE on an installed tool,
# available updates, a leaked secret, or a newer opsforge. It reads
# opsforge's cached digest (never the network) and refreshes a stale cache
# in the background, so opening a shell stays instant. Shows at most once
# per session.
#
# Disable with OPSFORGE_NOTIFY=0.

[[ "$OPSFORGE_NOTIFY" == "0" ]] && return
(( $+commands[opsforge] )) || return
[[ -o interactive ]] || return
[[ -n "$_OPSFORGE_NOTIFY_SHOWN" ]] && return
typeset -g _OPSFORGE_NOTIFY_SHOWN=1

# `notify --quiet` prints a compact one-liner only when there's something
# to report (empty otherwise), and refreshes a stale cache in the
# background. Run detached so shell startup never waits.
{
  local _line
  _line="$(opsforge notify --quiet 2>/dev/null)"
  [[ -n "$_line" ]] && print -r -- "$_line"
} &!
