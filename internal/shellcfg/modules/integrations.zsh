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

# opsenv: unlock the encrypted env vault into THIS shell. It must run in the
# current shell (not a subprocess) so the exports apply — hence a function, not
# an alias to `opsforge env load`. The passphrase prompt is written to stderr
# by `env load`, so command substitution captures only the export lines.
opsenv() {
  command -v opsforge >/dev/null 2>&1 || { print -u2 "opsenv: opsforge not found"; return 1; }
  local lines
  lines="$(opsforge env load)" || return $?   # prompts on stderr; exit code propagates
  eval "$lines"
  print -u2 "opsenv: vault loaded into this shell"
}
