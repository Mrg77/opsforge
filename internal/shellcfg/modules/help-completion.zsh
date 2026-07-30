# opsforge dynamic subcommand completion — TAB completion for tools that ship
# NO native zsh completion, by parsing their own `--help` on the fly. Same idea
# as the `?` inline help, but for TAB: the tool itself is the source of truth,
# so it's always current and needs no hand-maintained subcommand lists.
#
# Scope: this ONLY targets tools known to lack a real zsh completer (terraform,
# tofu…). Tools with native completion (kubectl, helm, gh, docker, git…) keep
# theirs — we never override a better completer.
#
# Cost control: `--help` forks a process, too slow to run on every keystroke, so
# results are cached per (tool + subcommand path) in a RAM file for a short TTL.
# Disable the whole thing with OPSFORGE_HELPCOMP=0.

[[ "$OPSFORGE_HELPCOMP" == "0" ]] && return

# _opsforge_help_parse reads a --help text on stdin and prints "name\tdesc" for
# each subcommand line. Handles the two dominant layouts:
#   terraform/kubectl/helm/docker:  "  name    description"   (2+ space gap)
#   gh:                             "  name:   description"   (colon after name)
# Flag lines (starting with '-') are skipped. Validated across those 5 tools.
_opsforge_help_parse() {
  awk '
    /^[[:space:]]+-/ { next }
    match($0, /^[[:space:]]+([a-z][a-z0-9_-]*):?[[:space:]][[:space:]]+[^[:space:]]/) {
      line=$0
      sub(/^[[:space:]]+/, "", line)
      name=line; sub(/[[:space:]:].*$/, "", name)
      desc=line; sub(/^[a-z0-9_-]*:?[[:space:]]+/, "", desc)
      if (length(name) > 0) print name "\t" desc
    }
  '
}

# _opsforge_help_subcommands <tool> <path...> — echo cached-or-fresh "name\tdesc"
# lines for `<tool> <path...> --help`. Cache key is the full command path; cache
# lives in $TMPDIR with a 1-hour TTL (help text rarely changes between runs).
_opsforge_help_subcommands() {
  local key cachedir cache ttl=3600
  cachedir="${TMPDIR:-/tmp}/opsforge-helpcomp-$UID"
  [[ -d "$cachedir" ]] || mkdir -p -m 700 "$cachedir" 2>/dev/null
  # Key: the whole argv, sanitized to a filename.
  key="${(j:_:)@}"
  key="${key//[^A-Za-z0-9_-]/-}"
  cache="$cachedir/$key"

  # Fresh cache within TTL? use it. (zsh/stat gives mtime cheaply.)
  if [[ -f "$cache" ]]; then
    local now mtime
    now=$EPOCHSECONDS
    zmodload -F zsh/stat b:zstat 2>/dev/null
    zstat -A mtime +mtime "$cache" 2>/dev/null
    if [[ -n "$mtime" ]] && (( now - mtime < ttl )); then
      print -r -- "$(<"$cache")"; return   # zsh native read, no external `cat`
    fi
  fi

  # Miss: run --help (stderr merged; some tools print help there), parse, cache.
  local out
  out="$("$@" --help 2>&1 | _opsforge_help_parse)"
  [[ -z "$out" ]] && out="$("$@" -h 2>&1 | _opsforge_help_parse)"
  print -r -- "$out" > "$cache" 2>/dev/null
  print -r -- "$out"
}

# _opsforge_help_complete — the completer. Offers subcommands for the current
# command path by parsing help; falls back to files when help yields nothing.
_opsforge_help_complete() {
  # Command path so far, minus the word being typed: words[1..CURRENT-1].
  local -a path
  path=(${words[1,CURRENT-1]})
  (( ${#path} )) || return

  local -a lines
  lines=(${(f)"$(_opsforge_help_subcommands $path)"})
  if (( ${#lines} )); then
    # _describe takes "name:description" entries (it splits on the first ':',
    # so a ':' inside a description is harmless). Our parser emits tab-separated
    # name/desc, so convert the tab to a colon.
    local -a specs
    local l
    for l in $lines; do
      specs+=("${l/$'\t'/:}")
    done
    _describe -t subcommands "${path[-1]} subcommand" specs && return
  fi
  _files
}

# Wire it onto tools that lack native zsh completion. Keep this list tight —
# only tools we've confirmed have no `<tool> completion zsh`.
for _oht in terraform tofu terragrunt packer; do
  command -v "$_oht" >/dev/null 2>&1 && compdef _opsforge_help_complete "$_oht"
done
unset _oht

# EPOCHSECONDS needs zsh/datetime; load it once (no-op if already loaded).
zmodload zsh/datetime 2>/dev/null
