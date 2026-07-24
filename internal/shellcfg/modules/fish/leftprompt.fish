# opsforge left prompt — a clean, informative prompt that never queries a
# cloud or a cluster (fish). Shows: the current directory (repo-relative in a
# git repo), the git branch with a dirty/ahead/behind marker, the duration of
# the last command when it was slow, and a ❯ that turns red on failure.
#
# opsforge never fights an existing prompt framework (starship, tide,
# oh-my-posh) and skips a fish_prompt you've defined yourself, unless forced
# with OPSFORGE_PROMPT=1. Disable entirely with OPSFORGE_PROMPT=0.
#
# fish gives us $CMD_DURATION (ms) and $status for free — no timing hooks
# needed, unlike zsh.

# fish always ships a default fish_prompt, so — unlike zsh — we can't tell a
# user's custom prompt from the stock one by inspecting it. opsforge therefore
# installs its prompt by default, and steps aside only for a known prompt
# framework or when explicitly disabled (OPSFORGE_PROMPT=0). A user who wants
# to keep their own prompt sets OPSFORGE_PROMPT=0.
if test "$OPSFORGE_PROMPT" != 0
    and test -z "$STARSHIP_SHELL$POWERLEVEL9K_MODE$POSH_THEME$TIDE_VERSION"

    function _opsforge_git_segment
        type -q git; or return
        set -l branch (git symbolic-ref --short HEAD 2>/dev/null; or git rev-parse --short HEAD 2>/dev/null)
        test -z "$branch"; and return

        set -l dirty ""
        git diff --quiet --ignore-submodules HEAD 2>/dev/null; or set dirty "*"
        # Untracked files add a "?" marker.
        if test -n (git ls-files --others --exclude-standard 2>/dev/null | head -1)
            set dirty "$dirty?"
        end

        set -l ab ""
        set -l counts (git rev-list --left-right --count HEAD...@{upstream} 2>/dev/null | string split -f1,2 \t)
        if test (count $counts) -eq 2
            test $counts[1] -gt 0 2>/dev/null; and set ab "$ab⇡$counts[1]"
            test $counts[2] -gt 0 2>/dev/null; and set ab "$ab⇣$counts[2]"
        end

        if test -n "$dirty"
            set_color yellow
        else
            set_color cyan
        end
        echo -n " $branch$dirty$ab"
        set_color normal
    end

    function _opsforge_duration_segment
        # $CMD_DURATION is milliseconds; only show noticeable ones (>= 2s).
        test -n "$CMD_DURATION"; or return
        test "$CMD_DURATION" -ge 2000 2>/dev/null; or return
        set -l secs (math "$CMD_DURATION / 1000")
        set_color brblack
        if test "$secs" -ge 60
            printf ' %dm%02ds' (math "floor($secs / 60)") (math "$secs % 60")
        else
            printf ' %.1fs' (math "$CMD_DURATION / 1000.0")
        end
        set_color normal
    end

    function fish_prompt
        set -l last $status

        # directory: repo-relative inside a git repo, else ~-shortened cwd.
        set -l root (git rev-parse --show-toplevel 2>/dev/null)
        set -l dir
        if test -n "$root"
            set dir (basename $root)(string replace -- $root '' $PWD)
        else
            set dir (string replace -- $HOME '~' $PWD)
        end

        set_color blue; echo -n $dir; set_color normal
        _opsforge_git_segment
        _opsforge_duration_segment
        echo ""

        # ❯ red on real failure; 130 (Ctrl-C) / 148 (Ctrl-Z) stay cyan.
        if test $last -ne 0 -a $last -ne 130 -a $last -ne 148
            set_color red
        else
            set_color cyan
        end
        echo -n "❯ "
        set_color normal
    end
end
