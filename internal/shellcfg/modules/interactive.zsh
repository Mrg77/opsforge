# opsforge interactive — a modern, Warp/Fish-like editing experience on
# top of your own zsh: as you type, a menu of matching commands/options
# appears automatically (no TAB), navigable with arrows or TAB, and you
# can keep typing to filter. Plus gray inline suggestions and syntax
# coloring.
#
# Powered by three battle-tested plugins, installed via brew during
# `shell install`:
#   - zsh-autocomplete       : the automatic, live completion menu
#   - zsh-autosuggestions    : gray inline suggestion (→ accepts)
#   - zsh-syntax-highlighting : green=valid / red=unknown command
#
# Disable entirely with OPSFORGE_INTERACTIVE=0.

[[ "$OPSFORGE_INTERACTIVE" == "0" ]] && return

_of_brew_share=""
if command -v brew >/dev/null 2>&1; then
  _of_brew_share="$(brew --prefix 2>/dev/null)/share"
fi

# --- zsh-autocomplete: automatic live menu as you type ------------------
# It manages compinit itself and MUST load before other completion widgets.
# A prior `compinit` in the user's ~/.zshrc doesn't prevent it from working
# because it re-initializes completion on load.
if [[ -n "$_of_brew_share" && -r "$_of_brew_share/zsh-autocomplete/zsh-autocomplete.plugin.zsh" ]]; then
  # Show the menu as soon as you type 1 character.
  zstyle ':autocomplete:*' min-input 1
  # Keep the list compact so it doesn't take over the screen.
  zstyle ':autocomplete:*' list-lines 8
  # TAB / arrows select from the menu; typing keeps filtering.
  zstyle ':autocomplete:tab:*' widget-style menu-select
  # Don't insert a match automatically — only when you pick one.
  zstyle ':autocomplete:*' insert-unambiguous yes

  # --- history-first menu ---------------------------------------------
  # The whole point: when you type `kubectl`, your recent *actual*
  # `kubectl …` command lines surface at the TOP of the live menu, above
  # the subcommand/flag completions — so you re-run what you did before
  # instead of retyping it. zsh-autocomplete already blends history into
  # the menu; we bias it toward real history and give it room.
  #   - reserve the first lines of the menu for history matches
  zstyle ':autocomplete:*' default-context history-incremental-search-backward
  zstyle ':autocomplete:history-search:*' list-lines 6
  #   - match anywhere in the line isn't what we want here: we want the
  #     line to START with what you typed (prefix), like fish.
  zstyle ':autocomplete:*' history-search-syntax prefix
  #   - keep the current session's commands weighted as most recent
  setopt SHARE_HISTORY INC_APPEND_HISTORY HIST_IGNORE_ALL_DUPS HIST_FIND_NO_DUPS

  source "$_of_brew_share/zsh-autocomplete/zsh-autocomplete.plugin.zsh"

  # Up-arrow does a prefix history search (type `kubectl`, press ↑ to walk
  # only your kubectl history) — the behavior most people expect.
  zstyle ':autocomplete:up:*'   fzf-completion no
  zstyle ':autocomplete:down:*' fzf-completion no
fi

# --- zsh-autosuggestions: gray inline suggestion as you type ---
if [[ -n "$_of_brew_share" && -r "$_of_brew_share/zsh-autosuggestions/zsh-autosuggestions.zsh" ]]; then
  # Suggest from your HISTORY (a whole past command line) rather than from
  # completion — so typing `kubectl` proposes a real previous command, not
  # a stray next token. Prefix-match keeps it relevant.
  ZSH_AUTOSUGGEST_STRATEGY=(history)
  ZSH_AUTOSUGGEST_HIGHLIGHT_STYLE='fg=8'
  # Don't fire on very long buffers (a pasted block shouldn't flicker).
  ZSH_AUTOSUGGEST_BUFFER_MAX_SIZE=80
  source "$_of_brew_share/zsh-autosuggestions/zsh-autosuggestions.zsh"
fi

# --- zsh-syntax-highlighting: color the command line (load LAST) ---
if [[ -n "$_of_brew_share" && -r "$_of_brew_share/zsh-syntax-highlighting/zsh-syntax-highlighting.zsh" ]]; then
  source "$_of_brew_share/zsh-syntax-highlighting/zsh-syntax-highlighting.zsh"
fi

unset _of_brew_share
