# Project Rules

## Documentation and Help Text

When changing any switched context type, profile field, environment variable, command behavior, install flow, or release flow, update every relevant user-facing surface in the same change:

- CLI help text in `internal/cli/root.go`.
- Sample config in `internal/config/config.go`.
- Runtime exports in `internal/switcher/switcher.go`.
- Tests in `internal/switcher/switcher_test.go`.
- README feature list, config example, switch behavior, and release instructions.
- npm metadata in `package.json` and platform package manifests.
- GitHub Release notes for the published tag.

Do not leave provider lists stale. Current switched context types are:

- AWS profile via `AWS_PROFILE`.
- Aliyun profile via `ALIBABA_CLOUD_PROFILE`.
- Codex profile via `CODEX_PROFILE`.
- SSH include target via `~/.config/opsctx/ssh-current`.

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
7. Wait for the `Release npm` workflow to pass.
8. Verify npm latest:

   ```sh
   npm view @ninj4dkill4/octx version
   ```

9. Create the GitHub Release for the tag.
10. Smoke test from npm:

   ```sh
   tmp=$(mktemp -d)
   npm install -g --prefix "$tmp" @ninj4dkill4/octx@latest
   "$tmp/bin/octx" --help
   ```

11. If README changed and npm web looks stale, verify registry readme directly:

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
