# opsforge integrations — wire up modern shell tools when present (bash).
# Each block is a no-op if its tool isn't installed.

# fzf: fuzzy finder key bindings (Ctrl-R history, Ctrl-T files) + completion.
if command -v fzf >/dev/null 2>&1; then
  if fzf --bash >/dev/null 2>&1; then
    eval "$(fzf --bash)"
  fi
fi

# zoxide: smarter `cd`. Provides `z` (jump) and `zi` (interactive).
if command -v zoxide >/dev/null 2>&1; then
  eval "$(zoxide init bash)"
fi

# atuin: full-text shell history with context. Loaded after fzf so its Ctrl-R
# binding wins.
if command -v atuin >/dev/null 2>&1; then
  eval "$(atuin init bash --disable-up-arrow)"
fi
