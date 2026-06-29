# octx

[![Release npm](https://github.com/ninj4dkill4/octx/actions/workflows/release-npm.yml/badge.svg)](https://github.com/ninj4dkill4/octx/actions/workflows/release-npm.yml)
[![npm version](https://img.shields.io/npm/v/%40ninj4dkill4%2Foctx.svg)](https://www.npmjs.com/package/@ninj4dkill4/octx)
[![license](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

`octx` is a small terminal context switcher for DevOps work. Pick a project once, then keep AWS, Codex, and SSH aligned in the current shell.

Phase 1 switches by project code only. Projects may still represent environments such as `dev`, `stg`, `uat`, and `prd`, but environment-level switching is intentionally left for a later phase.

## Features

- Fast project picker in the terminal.
- Exports `OPSCTX_PROJECT`.
- Exports or unsets `AWS_PROFILE` and `CODEX_PROFILE`.
- Updates or clears an SSH include symlink for the selected project.
- Stores the last selected project in local state.
- Ships as an npm package with native Go binaries for Linux and macOS.

## Install

```sh
npm install -g @ninj4dkill4/octx
```

Verify the install:

```sh
octx --help
```

## Shell Integration

`octx` needs a shell wrapper because a child process cannot export environment variables into its parent shell.

Add this to `~/.zshrc`:

```zsh
octx() {
  if [[ $# -eq 0 ]]; then
    eval "$(command octx --shell)"
  else
    command octx "$@"
  fi
}
```

If you use Codex profiles, add this wrapper too:

```zsh
codex() {
  if [[ -n "${CODEX_PROFILE:-}" ]]; then
    command codex --profile "$CODEX_PROFILE" "$@"
  else
    command codex "$@"
  fi
}
```

Reload your shell:

```sh
source ~/.zshrc
```

## Quick Start

Create a sample config:

```sh
octx init
```

Edit the generated config:

```text
~/.config/opsctx/config.yaml
```

Example:

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

Only `code` is required. `aws_profile`, `codex_profile`, and `ssh_config` are optional. If an optional profile is omitted, `octx` unsets the matching environment variable during switch. If `ssh_config` is omitted, `octx` removes the generated SSH include target.

Add this once to `~/.ssh/config`:

```sshconfig
Include ~/.config/opsctx/ssh-current
```

Switch context:

```sh
octx
```

Check the current project:

```sh
octx current
```

## What Switch Does

After selecting a project, `octx`:

- exports `OPSCTX_PROJECT`
- exports or unsets `AWS_PROFILE`
- exports or unsets `CODEX_PROFILE`
- writes `~/.config/opsctx/state.yaml`
- updates `~/.config/opsctx/ssh-current` to point to the configured project SSH config, or removes it when no `ssh_config` is configured

`CODEX_PROFILE` is intentionally just an environment variable. The `codex` shell wrapper maps it to:

```sh
codex --profile "$CODEX_PROFILE"
```

## Files

Default paths:

| Purpose | Path |
| --- | --- |
| Config | `~/.config/opsctx/config.yaml` |
| State | `~/.config/opsctx/state.yaml` |
| Current SSH include | `~/.config/opsctx/ssh-current` |

The config and state directory name is still `opsctx` for backward compatibility with early local installs.

## Commands

```sh
octx          # open picker and switch context
octx init     # write a sample config
octx current  # print the current project code
```

## Release

This repository publishes npm packages through GitHub Actions and npm Trusted Publishing.

Packages:

- `@ninj4dkill4/octx`
- `@ninj4dkill4/octx-linux-x64`
- `@ninj4dkill4/octx-linux-arm64`
- `@ninj4dkill4/octx-darwin-x64`
- `@ninj4dkill4/octx-darwin-arm64`

Publish a new version by pushing a semver tag:

```sh
git tag vX.Y.Z
git push origin vX.Y.Z
```

The workflow builds native binaries, prepares package manifests from the tag version, runs dry-run packs, and publishes to npm.

## Roadmap

- Environment-level switching.
- Kubeconfig support.
- Terraform workspace or variable support.
- Vault or secrets manager integration.
- Directory-aware auto-switching.

## License

MIT
