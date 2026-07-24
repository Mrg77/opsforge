# opsforge interactive layer (fish).
#
# fish ships the features the zsh layer has to bolt on via plugins — inline
# autosuggestions, syntax highlighting, and ↑ history search by the prefix
# you've typed — all built in. So there's almost nothing to install here; this
# module just makes the defaults explicit and honors OPSFORGE_INTERACTIVE=0.
#
# Disable the whole interactive layer with OPSFORGE_INTERACTIVE=0.

if test "$OPSFORGE_INTERACTIVE" != 0
    and status is-interactive

    # ↑/↓ already search history by the typed prefix in fish (up-or-search).
    # → (forward-char) accepts the autosuggestion at end of line. These are
    # fish defaults; we bind them explicitly so the behavior is guaranteed
    # even under a non-default key-bindings preset.
    bind up up-or-search
    bind down down-or-search
end
