# opsforge interactive — a modern editing experience on top of your own
# zsh, using battle-tested plugins. Chosen for reliability over flashiness:
# zsh-autocomplete is intentionally NOT used here because it fights any
# existing compinit in the user's .zshrc and breaks TAB. Instead we use:
#
#   - zsh-autosuggestions    : gray inline suggestion as you type (→ accepts)
#   - a navigable TAB menu    : press TAB to open a completion menu you move
#                               through with arrows / TAB (native zsh, robust)
#   - zsh-syntax-highlighting : colors the command line (green=valid, red=unknown)
#
# Each block is a no-op when its plugin is absent, so this file is safe to
# source anywhere. Disable the whole thing with OPSFORGE_INTERACTIVE=0.

[[ "$OPSFORGE_INTERACTIVE" == "0" ]] && return

# Make sure the completion system is initialized (it usually already is,
# but this guarantees TAB completion works even on a bare .zshrc).
if ! typeset -f compdef >/dev/null 2>&1; then
  autoload -Uz compinit && compinit -u
fi

# --- A rich, navigable TAB menu -----------------------------------------
# One TAB shows the menu; further TABs / arrows move through it. This is
# the reliable way to get "show me everything this command can do".
zmodload zsh/complist 2>/dev/null
zstyle ':completion:*' menu select                      # arrow-navigable menu
zstyle ':completion:*' group-name ''                    # group by type
zstyle ':completion:*:descriptions' format '%F{cyan}%d%f'
zstyle ':completion:*' matcher-list 'm:{a-zA-Z}={A-Za-z}' # case-insensitive
zstyle ':completion:*' list-colors "${(s.:.)LS_COLORS}"  # colorful entries
setopt AUTO_MENU        # show the menu on a second consecutive tab
setopt COMPLETE_IN_WORD # complete from the cursor, not just end of word
setopt ALWAYS_TO_END    # move cursor to end after a completion

_of_brew_share=""
if command -v brew >/dev/null 2>&1; then
  _of_brew_share="$(brew --prefix 2>/dev/null)/share"
fi

# --- zsh-autosuggestions: gray inline suggestion as you type ---
if [[ -n "$_of_brew_share" && -r "$_of_brew_share/zsh-autosuggestions/zsh-autosuggestions.zsh" ]]; then
  ZSH_AUTOSUGGEST_HIGHLIGHT_STYLE='fg=8'          # dim gray
  ZSH_AUTOSUGGEST_STRATEGY=(history completion)   # history first, then completions
  source "$_of_brew_share/zsh-autosuggestions/zsh-autosuggestions.zsh"
  # Accept the whole suggestion with the right arrow.
  bindkey '^[[C' forward-char
fi

# --- zsh-syntax-highlighting: color the command line (load LAST) ---
if [[ -n "$_of_brew_share" && -r "$_of_brew_share/zsh-syntax-highlighting/zsh-syntax-highlighting.zsh" ]]; then
  source "$_of_brew_share/zsh-syntax-highlighting/zsh-syntax-highlighting.zsh"
fi

unset _of_brew_share
