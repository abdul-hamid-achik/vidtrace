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

- darwin and linux archives for amd64 and arm64 (macOS binaries signed + notarized)
- Linux `.deb` and `.rpm` packages for amd64 and arm64
- `checksums.txt`
- GitHub release notes
- `Casks/vidtrace.rb` in the tap

## Distribution decisions

- **Homebrew: cask, not formula.** `vidtrace` ships a prebuilt binary, so the cask installs it directly without a build step. A formula would only duplicate that for a CLI, so it is intentionally not added. Now that the binary is signed and notarized, the cask no longer needs the old quarantine workaround.
- **Linux: `.deb` + `.rpm` via nfpms**, attached to each release. No system dependencies are declared (see `docs/INSTALL.md`); `vidtrace doctor` reports the runtime tools.

## macOS signing and notarization

macOS release binaries are signed with a Developer ID certificate and notarized by Apple via GoReleaser's `notarize.macos` block, which uses the bundled `quill` (no external tool is installed on the Linux runner). Signing is gated on `isEnvSet "MACOS_SIGN_P12"`, so a build without the secrets still succeeds unsigned.

Required repository secrets:

- `MACOS_SIGN_P12` — base64-encoded Developer ID Application `.p12` certificate
- `MACOS_SIGN_PASSWORD` — the `.p12` export password
- `APPLE_API_ISSUER_ID`, `APPLE_API_KEY_ID` — App Store Connect API key Issuer and Key IDs
- `APPLE_API_KEY` — the raw `.p8` private key (the workflow base64-encodes it into `NOTARY_KEY_B64` for GoReleaser)

To create the credentials: make a Developer ID Application certificate (via a Keychain CSR or Xcode → Manage Certificates), export it as a `.p12`, and generate an App Store Connect API key under Users and Access → Integrations → Keys. Verify the signing identity locally with `security find-identity -v -p codesigning`.

## One-Time Setup

The public repository is `https://github.com/abdul-hamid-achik/vidtrace`. The release workflow uses these repository secrets:

- `HOMEBREW_TAP_TOKEN`: a token with write access to `abdul-hamid-achik/homebrew-tap` (required)
- `MACOS_SIGN_P12`, `MACOS_SIGN_PASSWORD`, `APPLE_API_ISSUER_ID`, `APPLE_API_KEY_ID`, `APPLE_API_KEY`: macOS signing + notarization (see above). If absent, the release still succeeds with unsigned macOS binaries.

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
