# opsforge aliases & functions — muscle-memory shortcuts and small OPS
# helpers (fish). Everything is guarded on the underlying tool being installed,
# so nothing shadows a command you don't have.

if command -q kubectl
    alias k 'kubectl'
    alias kg 'kubectl get'
    alias kd 'kubectl describe'
    alias kl 'kubectl logs'
    alias kga 'kubectl get all'

    # kx: switch kube context (fzf picker when available, else list/set).
    function kx
        if test -n "$argv[1]"
            kubectl config use-context "$argv[1]"
            return
        end
        if command -q fzf
            set -l ctx (kubectl config get-contexts -o name | fzf --height 40% --prompt='context> ')
            test -n "$ctx"; and kubectl config use-context "$ctx"
        else
            kubectl config get-contexts
        end
    end

    # kn: switch namespace for the current context.
    function kn
        if test -n "$argv[1]"
            kubectl config set-context --current --namespace="$argv[1]"
            return
        end
        if command -q fzf
            set -l ns (kubectl get ns -o name 2>/dev/null | sed 's|namespace/||' | fzf --height 40% --prompt='namespace> ')
            test -n "$ns"; and kubectl config set-context --current --namespace="$ns"
        else
            kubectl get ns
        end
    end
end

command -q terraform; and begin; alias tf 'terraform'; alias tfp 'terraform plan'; end
command -q docker; and alias dc 'docker compose'
command -q helm; and alias h 'helm'
command -q git; and begin; alias gst 'git status'; alias gd 'git diff'; end
command -q bat; and alias cat 'bat --paging=never'
command -q eza; and alias ls 'eza'

# `hg <term>` — grep your whole history fast. For a DevOps-tool view,
# `opsforge history kube` groups it by family (kube/git/tf/docker/cloud…).
# fish's own `history` builtin already shows a useful, searchable view.
function hg
    history search --contains -- $argv
end
