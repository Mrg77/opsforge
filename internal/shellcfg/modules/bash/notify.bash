# opsforge notifications — a one-time, ambient heads-up on shell start (bash)
# when something needs attention: a CVE on an installed tool, available
# updates, a leaked secret, or a newer opsforge. Reads opsforge's cached digest
# (never the network) and refreshes a stale cache in the background, so opening
# a shell stays instant. Shows at most once per session.
#
# Disable with OPSFORGE_NOTIFY=0.

case $- in *i*) ;; *) return 0 2>/dev/null || true ;; esac

if [[ "$OPSFORGE_NOTIFY" != "0" ]] \
   && command -v opsforge >/dev/null 2>&1 \
   && [[ -z "$_OPSFORGE_NOTIFY_SHOWN" ]]; then
  _OPSFORGE_NOTIFY_SHOWN=1
  # `notify --quiet` prints a compact one-liner only when there's something to
  # report, and refreshes a stale cache in the background. Detached so shell
  # startup never waits.
  {
    _of_line="$(opsforge notify --quiet 2>/dev/null)"
    [[ -n "$_of_line" ]] && printf '%s\n' "$_of_line"
    unset _of_line
  } &
  disown 2>/dev/null || true
fi
