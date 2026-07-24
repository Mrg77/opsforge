# opsforge integrations — wire up modern shell tools when present (fish).
# Each block is a no-op if its tool isn't installed, so this file is safe to
# source on any machine.

# fzf: fuzzy finder key bindings (Ctrl-R history, Ctrl-T files) + completion.
if command -q fzf
    fzf --fish 2>/dev/null | source
end

# zoxide: smarter `cd`. Provides `z` (jump) and `zi` (interactive).
if command -q zoxide
    zoxide init fish | source
end

# atuin: full-text shell history with context. Loaded after fzf so its Ctrl-R
# binding wins (atuin's search UI is the better one).
if command -q atuin
    atuin init fish --disable-up-arrow | source
end
