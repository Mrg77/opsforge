# opsforge integrations — wire up modern shell tools when present. Each
# block is a no-op if its tool is not installed, so this file is safe to
# source on any machine.

# fzf: fuzzy finder key bindings (Ctrl-R history, Ctrl-T files) + completion.
if command -v fzf >/dev/null 2>&1; then
  if fzf --zsh >/dev/null 2>&1; then
    source <(fzf --zsh)
  fi
fi

# zoxide: smarter `cd`. Provides `z` (jump) and `zi` (interactive).
if command -v zoxide >/dev/null 2>&1; then
  eval "$(zoxide init zsh)"
fi

# atuin: full-text shell history with context. Loaded after fzf so its
# Ctrl-R binding wins (atuin's search UI is the better one).
if command -v atuin >/dev/null 2>&1; then
  eval "$(atuin init zsh --disable-up-arrow)"
fi
