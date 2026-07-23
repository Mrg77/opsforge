# opsforge CVE notice — a one-time, ambient heads-up when a tool on your
# machine has a known CVE. It reads opsforge's cached scan (never the
# network) and kicks a background refresh when stale, so opening a shell
# stays instant. Shows at most once per shell session.
#
# Disable with OPSFORGE_CVE_NOTICE=0.

[[ "$OPSFORGE_CVE_NOTICE" == "0" ]] && return
(( $+commands[opsforge] )) || return

# Only in interactive shells, and only once per session.
[[ -o interactive ]] || return
[[ -n "$_OPSFORGE_CVE_SHOWN" ]] && return
typeset -g _OPSFORGE_CVE_SHOWN=1

# `cve check` prints a short line only when there's something to report and
# a cache exists; empty output = stay quiet. It also refreshes a stale
# cache in the background. Run it detached so shell startup never waits.
{
  local _line
  _line="$(opsforge cve check --refresh-stale 2>/dev/null)"
  [[ -n "$_line" ]] && print -r -- "$_line"
} &!
