# opsforge special completions — tools that don't emit a completion
# script but register a completer via the bash-style `complete -C`
# directive (terraform, opentofu, aws...). zsh can use these through
# bashcompinit. Each block is a no-op when the tool is absent.

# bashcompinit lets zsh honor bash `complete` directives.
if ! typeset -f complete >/dev/null 2>&1; then
  autoload -Uz +X bashcompinit 2>/dev/null && bashcompinit 2>/dev/null
fi

# terraform: `complete -C terraform terraform`
if command -v terraform >/dev/null 2>&1 && typeset -f complete >/dev/null 2>&1; then
  complete -o nospace -C terraform terraform 2>/dev/null
fi

# opentofu: same mechanism, binary is `tofu`
if command -v tofu >/dev/null 2>&1 && typeset -f complete >/dev/null 2>&1; then
  complete -o nospace -C tofu tofu 2>/dev/null
fi

# aws: ships a dedicated completer binary
if command -v aws_completer >/dev/null 2>&1 && typeset -f complete >/dev/null 2>&1; then
  complete -C aws_completer aws 2>/dev/null
fi
