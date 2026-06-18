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

## First Release

One-time setup before the first tag:

```bash
git add .
git commit -m "Initial vidtrace CLI"
git remote add origin git@github.com:abdul-hamid-achik/vidtrace.git
git push -u origin main
```

Configure the `HOMEBREW_TAP_TOKEN` repository secret before pushing the release tag.

```bash
task check
goreleaser check
git tag v0.1.0
git push origin v0.1.0
```

GoReleaser creates:

- darwin and linux archives for amd64 and arm64
- `checksums.txt`
- GitHub release notes
- `Casks/vidtrace.rb` in the tap

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
