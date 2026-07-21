# opsforge interactive — a modern, Fish/Warp-like editing experience on
# top of your own zsh, using three battle-tested plugins:
#
#   - zsh-autosuggestions   : gray inline suggestion as you type (→ accepts)
#   - zsh-autocomplete      : a completion menu that appears automatically,
#                             no TAB needed
#   - zsh-syntax-highlighting: colors the command line (green=valid, red=unknown)
#
# opsforge installs these via brew during `shell install`. Each block is a
# no-op when its plugin is absent, so this file is safe to source anywhere.
# Disable the whole thing with OPSFORGE_INTERACTIVE=0.

[[ "$OPSFORGE_INTERACTIVE" == "0" ]] && return

# Resolve the brew prefix once (works on Apple Silicon and Intel).
_of_brew_share=""
if command -v brew >/dev/null 2>&1; then
  _of_brew_share="$(brew --prefix 2>/dev/null)/share"
fi

# --- zsh-autocomplete: automatic menu under the cursor (load FIRST) ---
# It manages compinit itself and must be sourced before other widgets.
if [[ -n "$_of_brew_share" && -r "$_of_brew_share/zsh-autocomplete/zsh-autocomplete.plugin.zsh" ]]; then
  # Keep it calm: only show the menu, don't insert the first match, and
  # keep TAB working the classic way too.
  zstyle ':autocomplete:*' min-input 1          # start suggesting after 1 char
  zstyle ':autocomplete:*' widget-style menu-select
  source "$_of_brew_share/zsh-autocomplete/zsh-autocomplete.plugin.zsh"
fi

# --- zsh-autosuggestions: gray inline suggestion as you type ---
if [[ -n "$_of_brew_share" && -r "$_of_brew_share/zsh-autosuggestions/zsh-autosuggestions.zsh" ]]; then
  ZSH_AUTOSUGGEST_HIGHLIGHT_STYLE='fg=8'        # dim gray
  ZSH_AUTOSUGGEST_STRATEGY=(history completion) # history first, then completion
  source "$_of_brew_share/zsh-autosuggestions/zsh-autosuggestions.zsh"
fi

# --- zsh-syntax-highlighting: color the command line (load LAST) ---
if [[ -n "$_of_brew_share" && -r "$_of_brew_share/zsh-syntax-highlighting/zsh-syntax-highlighting.zsh" ]]; then
  source "$_of_brew_share/zsh-syntax-highlighting/zsh-syntax-highlighting.zsh"
fi

unset _of_brew_share
