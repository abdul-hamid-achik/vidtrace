# Release

`vidtrace` uses GitHub Actions and GoReleaser for tagged releases.

## CI

`.github/workflows/ci.yml` runs on pushes, pull requests, and manual dispatch:

- formatting check with `gofmt`
- `go mod tidy` drift check
- `go test ./...`
- binary build
- golangci-lint `v2.12.2`
- `goreleaser check`

CI does not run Whisper or OCR extraction because those checks require large runtime dependencies and media fixtures. Use `task all` locally for the full media-path verification.

## Release Workflow

`.github/workflows/release.yml` runs on `v*` tags and manual dispatch.

Required repository secret:

- `HOMEBREW_TAP_TOKEN`: a token with write access to `abdul-hamid-achik/homebrew-tap`

The normal `GITHUB_TOKEN` publishes the GitHub release in this repository. The Homebrew tap token updates the separate tap repository.

## Routine Release

Before tagging, update `CHANGELOG.md` and run the local checks:

```bash
task all
goreleaser check
```

Tag the release from `main`:

```bash
git tag vX.Y.Z
git push origin vX.Y.Z
```

GoReleaser creates:

- darwin and linux archives for amd64 and arm64
- Linux `.deb` and `.rpm` packages for amd64 and arm64
- `checksums.txt`
- GitHub release notes
- `Casks/vidtrace.rb` in the tap

## Distribution decisions

- **Homebrew: cask, not formula.** `vidtrace` ships a prebuilt binary, so the cask installs it directly without a build step. A formula would only duplicate that for a CLI, so it is intentionally not added. The cask currently strips the macOS quarantine attribute as a workaround for the unsigned binary; once signing and notarization are in place (below), that workaround and the security caveat can be removed.
- **Linux: `.deb` + `.rpm` via nfpms**, attached to each release. No system dependencies are declared (see `docs/INSTALL.md`); `vidtrace doctor` reports the runtime tools.

## macOS signing and notarization (playbook)

The binary is currently unsigned, so macOS Gatekeeper quarantines it (the cask works around this with `xattr`). Proper Developer ID signing + notarization removes the warning. This requires credentials that must be provided before it can be wired up:

Prerequisites (maintainer):

1. Enroll in the **Apple Developer Program** (paid; an Apple ID alone is not enough). Enrollment can take a few days.
2. Create a **Developer ID Application** certificate and export it as a password-protected `.p12`.
3. Create an **App Store Connect API key** (Issuer ID, Key ID, and the `.p8` private key) for `notarytool`.

GitHub repository secrets to add once obtained:

- `MACOS_SIGN_P12` — base64-encoded `.p12` certificate
- `MACOS_SIGN_PASSWORD` — the `.p12` export password
- `APPLE_API_ISSUER_ID`, `APPLE_API_KEY_ID`, `APPLE_API_KEY` — App Store Connect API key fields

When those exist, the release will sign and notarize the macOS binaries (using `rcodesign` so it runs on the existing Linux runner), gated so a release without the secrets still succeeds unsigned, and the cask's quarantine workaround and caveat will be dropped. Until then, releases ship unsigned and the cask keeps the workaround.

## One-Time Setup

The public repository is `https://github.com/abdul-hamid-achik/vidtrace`. The release workflow needs this repository secret before publishing tags:

- `HOMEBREW_TAP_TOKEN`: a token with write access to `abdul-hamid-achik/homebrew-tap`

## Homebrew Install

After the tag workflow passes:

```bash
brew tap abdul-hamid-achik/tap
brew install --cask abdul-hamid-achik/tap/vidtrace
vidtrace doctor
```

## Local Snapshot

Use a snapshot to inspect release artifacts without publishing:

```bash
goreleaser release --snapshot --clean --skip publish
```

The generated files stay under `dist/`, which is ignored by Git.

## Notes

- Do not store release tokens in the repo.
- Do not commit generated `dist/` output.
- Update `.goreleaser.yaml` if the final GitHub repository owner differs from the module path.
