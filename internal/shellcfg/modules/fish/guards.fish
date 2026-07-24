# opsforge guards — policy-as-code for destructive commands (fish).
#
# Implemented by rebinding Enter to a function that inspects the command line
# before running it. `commandline -f execute` is fish's official way to run the
# current line, so a safe command behaves exactly as usual; a guarded one is
# intercepted first. This mirrors the zsh accept-line widget.
#
# The decision is made by `opsforge guard check`, which reads the context
# passively — it NEVER invokes kubectl, so it can't trigger an OIDC browser
# login. Set OPSFORGE_GUARDS=0 to disable for a session.

# Derive the cheap prefilter once, from the ACTIVE policy, so a custom rule is
# never silently skipped. Override with OPSFORGE_GUARD_PREFILTER.
if not set -q OPSFORGE_GUARD_PREFILTER
    if type -q opsforge
        set -g OPSFORGE_GUARD_PREFILTER (opsforge guard prefilter 2>/dev/null)
    end
end
if test -z "$OPSFORGE_GUARD_PREFILTER"
    set -g OPSFORGE_GUARD_PREFILTER '(kubectl|helm|terraform|kubens|kubectx|argocd|flux|k)'
end

# _opsforge_looks_dangerous: a zero-subprocess gate. The prefilter is a
# regex-alternation like "(kubectl|helm|…)"; test the lowercased buffer against
# it so most commands skip the Go call entirely.
function _opsforge_looks_dangerous
    string match -qri -- $OPSFORGE_GUARD_PREFILTER (string lower -- $argv[1])
end

function _opsforge_guard_enter
    set -l buf (commandline)

    if test "$OPSFORGE_GUARDS" != 0
        and type -q opsforge
        and test -n "$buf"
        and _opsforge_looks_dangerous "$buf"

        # `guard check` reads the context itself. Output is "action" or
        # "action|message".
        set -l reply (opsforge guard check "$buf" 2>/dev/null)
        set -l action (string split -m1 '|' -- $reply)[1]
        set -l message (string split -m1 '|' -- $reply)[2]

        switch "$action"
            case deny
                echo ""
                set_color -o red; echo "✗  Blocked by opsforge guard"; set_color normal
                test -n "$message"; and begin; set_color red; echo "   $message"; set_color normal; end
                set_color yellow; echo "   $buf"; set_color normal
                set_color brblack; echo "   (disable guards for this session with OPSFORGE_GUARDS=0)"; set_color normal
                commandline ""
                commandline -f repaint
                return
            case warn
                echo ""
                set_color -o yellow
                if test -n "$message"; echo "⚠  $message"; else; echo "⚠  opsforge guard"; end
                set_color normal
                # warn does not stop the command; fall through to execute.
            case confirm
                echo ""
                set_color -o red; echo "⚠  opsforge guard"; set_color normal
                test -n "$message"; and begin; set_color red; echo "   $message"; set_color normal; end
                set_color yellow; echo "   $buf"; set_color normal
                set_color brblack; echo "   (to skip guards this session: OPSFORGE_GUARDS=0)"; set_color normal
                read -l -P "Type 'yes' to run this: " answer
                if test "$answer" != yes
                    set_color red; echo "Aborted by opsforge guard."; set_color normal
                    commandline ""
                    commandline -f repaint
                    return
                end
        end
    end

    commandline -f execute
end

if status is-interactive
    bind enter _opsforge_guard_enter
end
