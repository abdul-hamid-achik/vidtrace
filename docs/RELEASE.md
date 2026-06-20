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

## macOS signing and notarization (blocked)

The binary is currently unsigned, so macOS Gatekeeper quarantines it (the cask works around this with `xattr`). Proper Developer ID signing + notarization would remove the warning. Enrollment and credentials are **done**, but signing is **blocked by an Apple-side certificate issue**, so releases still ship unsigned.

What is in place:

- Apple Developer Program enrollment.
- A **Developer ID Application** certificate, exported as a password-protected `.p12`.
- An **App Store Connect API key** (Issuer ID, Key ID, and the `.p8` private key) for notarization.
- GitHub repository secrets: `MACOS_SIGN_P12` (base64 `.p12`), `MACOS_SIGN_PASSWORD`, `APPLE_API_ISSUER_ID`, `APPLE_API_KEY_ID`, `APPLE_API_KEY`.

### The blocker

A GoReleaser `notarize.macos` block (which uses the bundled `quill`) was wired up and the release failed during signing with:

```
failed to verify certificate chain: x509: unhandled critical extension
```

Inspecting the certificate confirms the cause: newly-issued Developer ID certificates mark the Apple extension OID `1.2.840.113635.100.6.1.13` as **critical**. Go's `crypto/x509` (which `quill` is built on) treats any unrecognized critical extension as a hard failure, so it cannot build the chain. This is an Apple/toolchain problem, not a configuration error, and GoReleaser's `notarize` offers no option to skip chain verification. The signing changes were reverted (commit history) so releases continue to publish unsigned.

### Options to re-enable later

- **`rcodesign`** (Rust `apple-codesign`): sign each darwin binary with `rcodesign sign` plus a separate `rcodesign notary-submit` step, instead of GoReleaser's quill-based `notarize`. Its X.509 parser may tolerate the critical extension. This is not native to GoReleaser OSS, so it needs custom workflow steps and is untested here.
- **Wait for `quill`/Go** to handle the extension, then restore the reverted `notarize.macos` block as-is.

Until one of those lands, releases ship unsigned and the cask keeps the `xattr` quarantine workaround and caveat.

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
