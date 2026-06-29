# octx

`octx` switches devops terminal context by project code.

Phase 1 switches:

- AWS profile
- Codex profile
- SSH config include target

Projects may have environments such as `dev`, `stg`, `uat`, and `prd`, but Phase 1 switches only by project code.

## Install

```bash
go install ./cmd/octx
```

Add the Go bin directory and shell wrappers to `~/.zshrc`:

```zsh
export PATH="$HOME/go/bin:$PATH"

octx() {
  if [[ $# -eq 0 ]]; then
    eval "$("$HOME/go/bin/octx" --shell)"
  else
    "$HOME/go/bin/octx" "$@"
  fi
}

codex() {
  if [[ -n "${CODEX_PROFILE:-}" ]]; then
    "$HOME/.local/bin/codex" --profile "$CODEX_PROFILE" "$@"
  else
    "$HOME/.local/bin/codex" "$@"
  fi
}
```

Reload the shell:

```bash
source ~/.zshrc
```

The `octx` wrapper is required because a child process cannot export environment variables into the current shell by itself.

The `codex` wrapper is required because Codex CLI uses `--profile <name>`; it does not read `CODEX_PROFILE` directly.

## Initialize Config

```bash
octx init
```

Default files:

- Config: `~/.config/opsctx/config.yaml`
- State: `~/.config/opsctx/state.yaml`
- Current SSH include: `~/.config/opsctx/ssh-current`

Example config:

```yaml
projects:
  - code: core
    name: Core Platform
    aws_profile: core-devops
    codex_profile: core
    ssh_config: ~/.ssh/config.d/core

  - code: pay
    name: Payment
    aws_profile: payment-devops
    codex_profile: payment
    ssh_config: ~/.ssh/config.d/payment
```

`codex_profile` maps to a Codex profile file:

```text
~/.codex/<codex_profile>.config.toml
```

For example, `codex_profile: core` uses:

```text
~/.codex/core.config.toml
```

## SSH Setup

Add this once to `~/.ssh/config`:

```sshconfig
Include ~/.config/opsctx/ssh-current
```

When switching project, `octx` updates `ssh-current` to point at the configured project SSH config.

## Usage

Switch project with the TUI:

```bash
octx
```

After selection, the shell wrapper exports:

```bash
OPSCTX_PROJECT=core
AWS_PROFILE=core-devops
CODEX_PROFILE=core
```

It also updates:

- State file: `~/.config/opsctx/state.yaml`
- SSH include target: `~/.config/opsctx/ssh-current`

After that, run Codex normally:

```bash
codex
```

The `codex` shell wrapper will call:

```bash
~/.local/bin/codex --profile "$CODEX_PROFILE"
```

Show current project:

```bash
octx current
```

Available user commands:

```bash
octx
octx init
octx current
```

## Phase 1 Limits

- No environment selection yet.
- No secrets management.
- No direct mutation of AWS credentials/config.
- No kubeconfig, Terraform, Vault, or directory auto-switch yet.
