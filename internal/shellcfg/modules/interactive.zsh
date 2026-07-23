# opsforge interactive — a calm, fish-like editing experience on top of
# your own zsh:
#   - press ↑ to walk your history filtered by what you've already typed
#     (prefix of the WHOLE line), Ctrl-R for a full search
#   - a gray inline suggestion from history you accept with →
#   - green/red syntax coloring as you type
#   - TAB completes; no menu is forced open on every keystroke
#
# By default the completion menu does NOT pop open on its own — typing
# stays quiet and history is on-demand (↑ / Ctrl-R). If you prefer the
# always-on live menu (zsh-autocomplete), set OPSFORGE_AUTOMENU=1.
#
# Disable this whole layer with OPSFORGE_INTERACTIVE=0.

[[ "$OPSFORGE_INTERACTIVE" == "0" ]] && return

# Locate a zsh plugin's entry file across the common install layouts, so
# the interactive layer works on Linux (apt/pacman/manual) too, not just
# Homebrew. Usage: _of_plugin zsh-autosuggestions zsh-autosuggestions.zsh
# echoes the first readable path found, or nothing.
_of_plugin() {
  local name="$1" file="$2" d
  local -a dirs
  if command -v brew >/dev/null 2>&1; then
    dirs+=("$(brew --prefix 2>/dev/null)/share/$name")
  fi
  dirs+=(
    "/usr/share/$name"                              # Debian/Ubuntu apt
    "/usr/share/zsh/plugins/$name"                  # Arch
    "/usr/local/share/$name"                        # manual /usr/local
    "/usr/share/zsh-$name"                          # some distros
    "$HOME/.zsh/$name"                              # manual clone
    "${ZDOTDIR:-$HOME}/.zsh/$name"
  )
  for d in $dirs; do
    if [[ -r "$d/$file" ]]; then print -r -- "$d/$file"; return 0; fi
  done
  return 1
}

# Resolve each plugin's path once (empty when absent — the block is skipped).
_of_autocomplete="$(_of_plugin zsh-autocomplete zsh-autocomplete.plugin.zsh)"
_of_autosuggest="$(_of_plugin zsh-autosuggestions zsh-autosuggestions.zsh)"
_of_highlight="$(_of_plugin zsh-syntax-highlighting zsh-syntax-highlighting.zsh)"

# History behavior shared by both modes: de-dupe so one command doesn't
# dominate, and make the current session's commands searchable right away.
setopt HIST_IGNORE_ALL_DUPS HIST_FIND_NO_DUPS INC_APPEND_HISTORY SHARE_HISTORY

if [[ "$OPSFORGE_AUTOMENU" == "1" && -n "$_of_autocomplete" ]]; then
  # --- opt-in: always-on live menu (zsh-autocomplete) -------------------
  zstyle ':autocomplete:*' min-input 1
  zstyle ':autocomplete:*' list-lines 8
  zstyle ':autocomplete:tab:*' widget-style menu-select
  zstyle ':autocomplete:*' insert-unambiguous yes
  source "$_of_autocomplete"
else
  # --- default: quiet, on-demand completion + prefix history search -----
  # Native zsh completion (TAB), initialized once. `-u` avoids the insecure
  # -directory prompt on shared machines.
  autoload -Uz compinit
  compinit -u 2>/dev/null
  # A single TAB shows a navigable menu; typing keeps it out of your way.
  zstyle ':completion:*' menu select
  zstyle ':completion:*' matcher-list 'm:{a-z}={A-Za-z}' # case-insensitive
  zstyle ':completion:*' list-colors ''

  # ↑ / ↓ search history by the prefix already on the line: type
  # `kubectl get pods -n s` then ↑ and you cycle ONLY commands that begin
  # that way — the whole line is the prefix, not the word under the cursor.
  autoload -Uz up-line-or-beginning-search down-line-or-beginning-search
  zle -N up-line-or-beginning-search
  zle -N down-line-or-beginning-search
  bindkey "$terminfo[kcuu1]" up-line-or-beginning-search   2>/dev/null
  bindkey "$terminfo[kcud1]" down-line-or-beginning-search 2>/dev/null
  # Also bind the common escape sequences, in case terminfo is unset.
  bindkey '^[[A' up-line-or-beginning-search
  bindkey '^[[B' down-line-or-beginning-search
fi

# --- zsh-autosuggestions: gray inline suggestion from history ---
if [[ -n "$_of_autosuggest" ]]; then
  # Suggest a whole PAST command line (history strategy), not a stray next
  # token — so the gray hint is a real command you can accept with →.
  ZSH_AUTOSUGGEST_STRATEGY=(history)
  ZSH_AUTOSUGGEST_HIGHLIGHT_STYLE='fg=8'
  # Don't fire on very long buffers (a pasted block shouldn't flicker).
  ZSH_AUTOSUGGEST_BUFFER_MAX_SIZE=80

  source "$_of_autosuggest"

  # --- Tab = accept the gray suggestion one word at a time ------------
  # Why not just bind Tab to `forward-word`? The plugin only wraps a
  # widget's partial-accept behavior lazily, on first keypress, and its
  # $POSTDISPLAY isn't reliably readable from a custom widget — so a naive
  # approach silently does nothing (the exact symptom: Tab not advancing).
  #
  # Instead we ask the plugin for the suggestion SYNCHRONOUSLY via its own
  # `_zsh_autosuggest_fetch_suggestion` (it fills a local `suggestion` from
  # the current $BUFFER), take its next shell word, and append it. This
  # doesn't depend on POSTDISPLAY or lazy wrapping, so it always fires.
  # With no suggestion, Tab falls back to normal completion.
  _opsforge_accept_word() {
    local suggestion
    if typeset -f _zsh_autosuggest_fetch_suggestion >/dev/null; then
      _zsh_autosuggest_fetch_suggestion "$BUFFER"   # sets $suggestion
    fi
    # The suggestion is the whole line; the part after BUFFER is the gray
    # tail. BUFFER may end mid-word (e.g. "ansible-play"), so the tail can
    # start with the rest of that word ("ook …").
    local tail="${suggestion#$BUFFER}"
    if [[ -n "$tail" ]]; then
      # Chunk = leading spaces (if any) + the next run of non-space chars
      # (finishes the current word / takes the next one) + the spaces that
      # follow it, so one Tab lands the cursor on the next argument.
      local lead="${tail%%[^[:space:]]*}"       # leading spaces
      local afterlead="${tail#$lead}"
      local wordpart="${afterlead%%[[:space:]]*}" # up to next space
      local afterword="${afterlead#$wordpart}"
      local trail="${afterword%%[^[:space:]]*}"   # trailing spaces
      local chunk="$lead$wordpart$trail"
      if [[ -n "$chunk" ]]; then
        BUFFER="$BUFFER$chunk"
        CURSOR=$#BUFFER
        zle autosuggest-fetch 2>/dev/null  # redraw remaining gray suggestion
      fi
    else
      zle expand-or-complete    # no suggestion → normal completion
    fi
  }
  zle -N _opsforge_accept_word
  bindkey '^I' _opsforge_accept_word   # Tab
fi

# --- zsh-syntax-highlighting: color the command line (load LAST) ---
if [[ -n "$_of_highlight" ]]; then
  source "$_of_highlight"
fi

unset _of_autocomplete _of_autosuggest _of_highlight
unfunction _of_plugin
