# opsforge special completions — native zsh completion for tools that
# don't ship a `completion zsh` command. terraform/opentofu only offer a
# bash-style `complete -C` completer, which the live auto-menu
# (zsh-autocomplete) won't run automatically because it forks a process
# on every keystroke. So we provide a STATIC native completion of their
# subcommands (no external call), which the auto-menu shows instantly,
# just like git/docker. Each block is a no-op when its tool is absent.

# Terraform / OpenTofu share the same subcommands.
_opsforge_tf_subcommands=(
  'init:Prepare your working directory for other commands'
  'validate:Check whether the configuration is valid'
  'plan:Show changes required by the current configuration'
  'apply:Create or update infrastructure'
  'destroy:Destroy previously-created infrastructure'
  'console:Try Terraform expressions at an interactive command prompt'
  'fmt:Reformat your configuration in the standard style'
  'force-unlock:Release a stuck lock on the current workspace'
  'get:Install or upgrade remote Terraform modules'
  'graph:Generate a Graphviz graph of the steps'
  'import:Associate existing infrastructure with a resource'
  'login:Obtain and save credentials for a remote host'
  'logout:Remove locally-stored credentials for a remote host'
  'metadata:Metadata related commands'
  'output:Show output values from your root module'
  'providers:Show the providers required for this configuration'
  'refresh:Update the state to match remote systems'
  'show:Show the current state or a saved plan'
  'state:Advanced state management'
  'taint:Mark a resource instance as not fully functional'
  'test:Execute integration tests for Terraform modules'
  'untaint:Remove the tainted state from a resource instance'
  'version:Show the current Terraform version'
  'workspace:Workspace management'
)

_opsforge_terraform() {
  if (( CURRENT == 2 )); then
    _describe -t commands 'terraform command' _opsforge_tf_subcommands
  else
    _files   # after the subcommand, complete file paths
  fi
}

if command -v terraform >/dev/null 2>&1; then
  compdef _opsforge_terraform terraform
fi
if command -v tofu >/dev/null 2>&1; then
  compdef _opsforge_terraform tofu
fi
