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

# Second-level subcommands for the terraform commands that have them. Keyed by
# the level-1 command word ($words[2]); anything not listed here falls back to
# file completion after its subcommand.
_opsforge_tf_state=(
  'list:List resources in the state'
  'show:Show a resource in the state'
  'mv:Move an item in the state'
  'rm:Remove instances from the state'
  'pull:Pull current state and output to stdout'
  'push:Update remote state from a local state file'
  'replace-provider:Replace provider in the state'
)
_opsforge_tf_workspace=(
  'list:List workspaces'
  'select:Select a workspace'
  'new:Create a new workspace'
  'delete:Delete a workspace'
  'show:Show the name of the current workspace'
)
_opsforge_tf_providers=(
  'lock:Write out dependency locks for the configured providers'
  'mirror:Save local copies of all required provider plugins'
  'schema:Show schemas for the providers used in the configuration'
)

_opsforge_terraform() {
  if (( CURRENT == 2 )); then
    _describe -t commands 'terraform command' _opsforge_tf_subcommands
    return
  fi
  if (( CURRENT == 3 )); then
    # Second level: offer this command's subcommands when it has any.
    case "$words[2]" in
      state)     _describe -t subcommands 'terraform state subcommand' _opsforge_tf_state; return ;;
      workspace) _describe -t subcommands 'terraform workspace subcommand' _opsforge_tf_workspace; return ;;
      providers) _describe -t subcommands 'terraform providers subcommand' _opsforge_tf_providers; return ;;
    esac
  fi
  _files   # deeper, or commands with no fixed subcommands: complete file paths
}

if command -v terraform >/dev/null 2>&1; then
  compdef _opsforge_terraform terraform
fi
if command -v tofu >/dev/null 2>&1; then
  compdef _opsforge_terraform tofu
fi
