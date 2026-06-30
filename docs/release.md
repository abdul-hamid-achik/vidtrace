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

Inspecting the certificate confirms the trigger: Developer ID certificates mark the Apple extension OID `1.2.840.113635.100.6.1.13` as **critical**, and Go's `crypto/x509` (which `quill` is built on) treats an unrecognized critical extension as a hard failure. This is **not** a problem with the certificate — that OID being critical is by-design on *every* Developer ID cert (global, non-removable; Apple's own US-issued certs carry it, confirmed on Apple Developer Forums thread 803047). **Reissuing the certificate will not help.**

A nuance worth knowing before re-attempting `quill`: the `quill` version GoReleaser pins *already strips* `1.2.840.113635.100.6.1.13` (and `.6.1.18`) from the leaf before verifying (anchore/quill since v0.5.0). So our failure was most likely a **missing Developer ID G2 intermediate in the exported `.p12`**, or a critical extension on a non-leaf cert — `quill` runs with `failWithoutFullChain=true` and Go's strictness either way. Re-exporting the `.p12` with the full chain (leaf + key + "Developer ID Certification Authority - G2") might rescue the `quill` path, but that is unverified.

### Recommended re-enable path: `rcodesign`

The lowest-risk fix (researched 2026-06-20, not yet implemented) is to replace `quill` with **`rcodesign`** (Rust `apple-codesign`, indygreg/apple-platform-rs), which runs on the **existing Linux runner**:

- Its `x509-certificate` crate has **no critical-extension rejection path** — it cannot reproduce the Go failure. (`rcodesign` itself even *generates* Developer ID certs with that OID marked critical and round-trips it.)
- It **bundles the Apple intermediates** (incl. Developer ID G2) and auto-embeds them, neutralizing the missing-intermediate failure mode.
- Caveat: no published third-party success report for this exact cert, so confidence rests on source-code analysis — prove it on a throwaway prerelease tag (or a local `rcodesign sign` on a darwin build) before trusting it.

Sketch (keep GoReleaser for build/archive/checksum/cask; remove `notarize.macos`; add post-build steps on `ubuntu-latest`):

```bash
# build the App Store Connect key JSON from the existing .p8 / issuer / key-id secrets
rcodesign encode-app-store-connect-api-key "$APPLE_API_ISSUER_ID" "$APPLE_API_KEY_ID" AuthKey.p8 key.json
# sign each darwin binary with the .p12 secret
rcodesign sign --p12-file cert.p12 --p12-password "$MACOS_SIGN_PASSWORD" --for-notarization ./vidtrace
# notarize (bare Mach-O binaries cannot be stapled; Gatekeeper does an online ticket check on first run)
rcodesign notary-submit --api-key-file key.json --wait ./vidtrace
```

Either use `indygreg/apple-code-sign-action@v1` (pin `rcodesign_version: 0.29.0`; inputs map directly onto the existing secrets) or the CLI above as a GoReleaser build hook. A macOS-runner native `codesign` + `xcrun notarytool` job is a viable fallback but is strictly more CI surgery (temp keychain, `set-key-partition-list`, explicit G2 intermediate install, sign-before-checksum ordering) for the same outcome.

Until this is wired up, releases ship unsigned and the cask keeps the `xattr` quarantine workaround and caveat. The user-facing impact is limited to direct GitHub-Release downloaders; Homebrew users are unaffected.

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
