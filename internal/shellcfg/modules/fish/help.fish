# opsforge inline help — press "?" at the end of a command line to see what it
# does, without leaving your line (fish). Runs the command's native --help,
# rendered cleanly (bat when available), then restores your prompt. Press "?"
# on an empty line for the opsforge cheat-sheet, "??" to explain the last
# command via AI. Disable with OPSFORGE_HELP=0.

if test "$OPSFORGE_HELP" != 0

    function _opsforge_help_colorize
        awk '
            /^[A-Za-z][A-Za-z &-]*:[[:space:]]*$/ { printf "\033[1;36m%s\033[0m\n", $0; next }
            /^[[:space:]]*#/                      { printf "\033[32m%s\033[0m\n", $0; next }
            /^[[:space:]]*--?[A-Za-z]/            { printf "\033[33m%s\033[0m\n", $0; next }
            { print }
        '
    end

    function _opsforge_help_render
        set -l title $argv[1]
        set -l body $argv[2]
        set -l width (math "min($COLUMNS, 100)" 2>/dev/null; or echo 80)
        set -l rule (string repeat -n $width ─)

        set_color cyan; echo $rule
        set_color -o cyan; echo "  ❯ $title --help"; set_color normal
        set -l gist (printf '%s\n' $body | awk 'NF && $0 !~ /^Usage/ && $0 !~ /^[A-Za-z]+:/ {print; exit}')
        test -n "$gist"; and begin; set_color brblack; echo "  $gist"; set_color normal; end
        set_color cyan; echo $rule; set_color normal

        if command -q bat
            printf '%s\n' $body | bat --style=plain --language=man --color=always --paging=never 2>/dev/null
            or printf '%s\n' $body | _opsforge_help_colorize
        else
            printf '%s\n' $body | _opsforge_help_colorize
        end
    end

    function _opsforge_help_panel
        if command -q opsforge
            opsforge shell help 2>/dev/null; and return
        end
        set_color -o brmagenta; echo "  opsforge shell"; set_color normal
        echo "  press ? for help, ?? to explain the last command"
    end

    function _opsforge_help_widget
        set -l buf (commandline)

        # "?" on an empty line — the opsforge cheat-sheet.
        if test -z "$buf"
            echo ""
            _opsforge_help_panel
            commandline -f repaint
            return
        end

        # "??" — buffer already holds one "?"; explain the last command via AI.
        if test "$buf" = "?"
            commandline ""
            echo ""
            opsforge explain --last
            commandline -f repaint
            return
        end

        # Only intercept "?" at end of a line whose first word is a real command.
        set -l cursor (commandline -C)
        if test "$cursor" -ne (string length -- "$buf")
            commandline -i '?'
            return
        end
        set -l parts (string split ' ' -- $buf)
        if not command -q $parts[1]
            commandline -i '?'
            return
        end

        # KUBECONFIG neutralized so a kubectl --help can't trigger cloud auth.
        set -l help (env KUBECONFIG=/dev/null $parts --help 2>&1)
        test -z "$help"; and set help (env KUBECONFIG=/dev/null $parts help 2>&1)

        echo ""
        if command -q less
            _opsforge_help_render "$buf" "$help" | env LESS='-FRXQ' less --prompt='  ↑↓ scroll · q to close help ' 2>/dev/null
        else
            _opsforge_help_render "$buf" "$help"
        end
        commandline -f repaint
    end

    # Track the last command + exit status so `??` / `opsforge explain --last`
    # know what to explain.
    function _opsforge_track_last --on-event fish_postexec
        set -l code $status
        set -l dir "$HOME/.cache/opsforge"
        test -d "$dir"; or mkdir -p "$dir" 2>/dev/null
        printf '%s\n' $code >"$dir/last-status" 2>/dev/null
        printf '%s\n' $argv >"$dir/last-cmd" 2>/dev/null
    end

    if status is-interactive
        bind '?' _opsforge_help_widget
    end
end
