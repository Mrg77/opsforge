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

_of_brew_share=""
if command -v brew >/dev/null 2>&1; then
  _of_brew_share="$(brew --prefix 2>/dev/null)/share"
fi

# History behavior shared by both modes: de-dupe so one command doesn't
# dominate, and make the current session's commands searchable right away.
setopt HIST_IGNORE_ALL_DUPS HIST_FIND_NO_DUPS INC_APPEND_HISTORY SHARE_HISTORY

if [[ "$OPSFORGE_AUTOMENU" == "1" && -n "$_of_brew_share" \
      && -r "$_of_brew_share/zsh-autocomplete/zsh-autocomplete.plugin.zsh" ]]; then
  # --- opt-in: always-on live menu (zsh-autocomplete) -------------------
  zstyle ':autocomplete:*' min-input 1
  zstyle ':autocomplete:*' list-lines 8
  zstyle ':autocomplete:tab:*' widget-style menu-select
  zstyle ':autocomplete:*' insert-unambiguous yes
  source "$_of_brew_share/zsh-autocomplete/zsh-autocomplete.plugin.zsh"
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
if [[ -n "$_of_brew_share" && -r "$_of_brew_share/zsh-autosuggestions/zsh-autosuggestions.zsh" ]]; then
  # Suggest a whole PAST command line (history strategy), not a stray next
  # token — so the gray hint is a real command you can accept with →.
  ZSH_AUTOSUGGEST_STRATEGY=(history)
  ZSH_AUTOSUGGEST_HIGHLIGHT_STYLE='fg=8'
  # Don't fire on very long buffers (a pasted block shouldn't flicker).
  ZSH_AUTOSUGGEST_BUFFER_MAX_SIZE=80
  # Tab, when it accepts one word of the suggestion, must extend the gray
  # text — register forward-word as a partial-accept widget BEFORE sourcing
  # the plugin so it wraps it.
  ZSH_AUTOSUGGEST_PARTIAL_ACCEPT_WIDGETS+=(forward-word)
  # → accepts the WHOLE gray suggestion; that's the plugin default.
  source "$_of_brew_share/zsh-autosuggestions/zsh-autosuggestions.zsh"

  # --- Tab = accept the gray suggestion word-by-word ------------------
  # The problem this solves: with a gray history suggestion showing, a
  # bare Tab used to run FILE completion (jumping `play` → `examples/`).
  # Instead: when a suggestion is present, Tab accepts just the next word
  # of it (`play` → `playbooks/`). When there is NO suggestion, Tab does
  # ordinary completion. → still accepts the whole line.
  _opsforge_tab() {
    if [[ -n "$POSTDISPLAY" ]]; then
      zle forward-word          # a gray suggestion is showing: take one word
    else
      zle expand-or-complete    # nothing suggested: normal completion
    fi
  }
  zle -N _opsforge_tab
  bindkey '^I' _opsforge_tab    # Tab
fi

# --- zsh-syntax-highlighting: color the command line (load LAST) ---
if [[ -n "$_of_brew_share" && -r "$_of_brew_share/zsh-syntax-highlighting/zsh-syntax-highlighting.zsh" ]]; then
  source "$_of_brew_share/zsh-syntax-highlighting/zsh-syntax-highlighting.zsh"
fi

unset _of_brew_share
