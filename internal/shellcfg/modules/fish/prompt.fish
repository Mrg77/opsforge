# opsforge prompt — optional context segments on the right prompt (fish).
#
# Mirrors prompt.zsh: only the kube-context segment is on by default, read
# cheaply from the kubeconfig FILE (never runs kubectl, so it can't trigger an
# OIDC login). Cloud and terraform segments are opt-in.
#
# Toggle with env vars (in ~/.config/fish/config.fish, before the eval line):
#   set -x OPSFORGE_PROMPT_KUBE 0    # disable the kube segment
#   set -x OPSFORGE_PROMPT_CLOUD 1   # enable the cloud segment (off by default)
#   set -x OPSFORGE_PROMPT_TF 1      # enable the terraform segment (off by default)

function _opsforge_kube_segment
    test "$OPSFORGE_PROMPT_KUBE" = 0; and return
    set -l cfg (string split -m1 ':' -- "$KUBECONFIG")[1]
    test -z "$cfg"; and set cfg "$HOME/.kube/config"
    test -r "$cfg"; or return
    set -l ctx (grep -m1 '^current-context:' "$cfg" 2>/dev/null | sed 's/current-context:[[:space:]]*//; s/["\x27]//g')
    test -z "$ctx"; and return
    if string match -q '*prod*' -- $ctx
        set_color -o red
    else
        set_color cyan
    end
    echo -n "⎈ $ctx"
    set_color normal
end

function _opsforge_cloud_segment
    test "$OPSFORGE_PROMPT_CLOUD" = 1; or return
    if test -n "$AWS_VAULT"
        set_color yellow; echo -n " ☁ aws:$AWS_VAULT"; set_color normal
    else if test -n "$AWS_PROFILE"
        set_color yellow; echo -n " ☁ aws:$AWS_PROFILE"; set_color normal
    end
end

function _opsforge_tf_segment
    test "$OPSFORGE_PROMPT_TF" = 1; or return
    test -d .terraform; or return
    set -l ws (cat .terraform/environment 2>/dev/null)
    test -n "$ws" -a "$ws" != default; and begin
        set_color magenta; echo -n " ⧉ tf:$ws"; set_color normal
    end
end

# Only claim the right prompt if the user (or another theme) hasn't defined one.
if not functions -q fish_right_prompt
    function fish_right_prompt
        _opsforge_kube_segment
        _opsforge_cloud_segment
        _opsforge_tf_segment
    end
end
