# opsforge notifications — a one-time, ambient heads-up on shell start (fish)
# when something needs attention: a CVE on an installed tool, available
# updates, a leaked secret, or a newer opsforge. Reads opsforge's cached digest
# (never the network) and refreshes a stale cache in the background, so opening
# a shell stays instant. Shows at most once per session.
#
# Disable with OPSFORGE_NOTIFY=0.

if test "$OPSFORGE_NOTIFY" != 0
    and command -q opsforge
    and status is-interactive
    and not set -q _OPSFORGE_NOTIFY_SHOWN

    set -g _OPSFORGE_NOTIFY_SHOWN 1

    # `notify --quiet` prints a compact one-liner only when there's something to
    # report (empty otherwise), and refreshes a stale cache in the background.
    # Run detached so shell startup never waits.
    fish -c '
        set -l line (opsforge notify --quiet 2>/dev/null)
        test -n "$line"; and printf "%s\n" "$line"
    ' &
    disown
end
