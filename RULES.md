# Project Rules

## Documentation and Help Text

When changing any switched context type, profile field, environment variable, command behavior, install flow, or release flow, update every relevant user-facing surface in the same change:

- CLI help text in `internal/cli/root.go`.
- Sample config in `internal/config/config.go`.
- Runtime exports in `internal/switcher/switcher.go`.
- Tests in `internal/switcher/switcher_test.go`.
- Doctor checks in `internal/doctor` when a switched context type or profile source changes.
- README feature list, config example, switch behavior, and release instructions.
- npm metadata in `package.json` and platform package manifests.
- Version output and build-time version injection when release behavior changes.
- GitHub Release notes for the published tag.

Project display fields such as `color` are optional unless explicitly stated otherwise. Missing display fields must keep default terminal styling and must not trigger doctor warnings.

Do not leave provider lists stale. Current switched context types are:

- AWS profile via `AWS_PROFILE`.
- Aliyun profile via `ALIBABA_CLOUD_PROFILE`.
- Codex profile via `CODEX_PROFILE`.
- GCloud configuration via `CLOUDSDK_ACTIVE_CONFIG_NAME`.
- Azure CLI config directory via `AZURE_CONFIG_DIR`.
- Kubeconfig via `KUBECONFIG`.
- SSH config via `OCTX_SSH_CONFIG` and generated per-project files under `~/.config/opsctx/ssh/`.

All switched context integrations are optional. Doctor may warn about missing optional profiles, CLIs, SSH config files, or kubeconfig files, but it must not treat those as errors or exit non-zero for them.

Do not use shared mutable state as active context. The active context is the current shell environment. `state.yaml` and `ssh-current` are legacy only and must not be read or written by switch behavior.

Do not mutate a global SSH symlink or require `Include ~/.config/opsctx/ssh-current`. When any project declares `ssh_config`, switch behavior must generate a per-project SSH config file and export `OCTX_SSH_CONFIG`. Doctor may warn if `~/.ssh/config` still includes the legacy `ssh-current` path, but it must not fail for missing legacy include wiring.

Doctor output must be grouped by `[global]` and project sections. New profile-specific checks must emit scoped project results instead of flat global rows.

Doctor should show CLI paths for configured CLI-backed profile integrations under `[global]`, because CLI binaries belong to the machine rather than an individual project. Optional environment variables that are correctly unset should be omitted instead of reported as `OK`; report env rows when a configured value matches or when a value needs attention.

The picker must keep an `unset` option at the bottom of the list. Selecting it clears all `octx`-managed environment variables in the current shell. The default picker cursor must prefer `OPSCTX_PROJECT` from the current shell when it points to a configured project; missing, unset, or unknown shell context selects `unset`.

Prefer neutral project wording in summaries and repository metadata, such as:

```text
cloud profiles, CLI profiles, and SSH
```

Use provider-specific names only where the behavior is explicitly provider-specific.

## Release Flow

Before tagging a release:

1. Choose the next semver version.
2. Update `package.json`, all root optional dependencies, and all `npm/*/package.json` versions to that next version in the release commit.
3. Make the code/docs change.
4. Run:

   ```sh
   go test ./...
   node scripts/build-npm.mjs
   node scripts/prepare-npm-packages.mjs
   npm pack --dry-run
   npm pack --dry-run ./npm/linux-x64
   npm pack --dry-run ./npm/linux-arm64
   npm pack --dry-run ./npm/darwin-x64
   npm pack --dry-run ./npm/darwin-arm64
   ```

5. Commit and push `main`.
6. Push the matching semver tag, for example `v0.1.7`.
7. Wait for the `Release npm` workflow to pass. The workflow publishes npm packages, verifies the npm version, smoke tests installing the exact tag version, and creates the GitHub Release.
8. If README changed and npm web looks stale, verify registry readme directly:

   ```sh
   npm view @ninj4dkill4/octx readme
    ```

## Shell Cache Note

After upgrading `octx`, users may need to refresh shell command hashing if their shell still runs an old binary:

```sh
hash -r
rehash
command -v octx
octx --help
```
