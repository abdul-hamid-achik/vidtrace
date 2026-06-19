# Install

## Runtime Requirements

`vidtrace` orchestrates local media tools. Install these before running extraction:

- `ffmpeg`
- `ffprobe`
- `tesseract`
- `whisper`

Run `vidtrace doctor` after installation. It checks binaries, Tesseract language data, and cached Whisper models.

## Homebrew

The Homebrew cask is published from tagged releases:

```bash
brew tap abdul-hamid-achik/tap
brew install --cask abdul-hamid-achik/tap/vidtrace
vidtrace version
vidtrace doctor
```

Upgrade to the latest published cask with:

```bash
brew update
brew upgrade --cask abdul-hamid-achik/tap/vidtrace
```

`vidtrace` is not signed with an Apple Developer certificate yet. If macOS blocks the first run, clear quarantine on the installed binary:

```bash
xattr -dr com.apple.quarantine /opt/homebrew/Caskroom/vidtrace/*/vidtrace
```

Install common runtime dependencies on macOS:

```bash
brew install ffmpeg tesseract
pipx install openai-whisper
```

Whisper downloads its model on first use. `vidtrace` defaults to the `small` model.

## Source Build

Development tool versions are pinned in `.tool-versions`.

```bash
git clone https://github.com/abdul-hamid-achik/vidtrace.git
cd vidtrace
task build
bin/vidtrace doctor
```

Required development tools:

- Go `1.26.4`
- Task `3.51.1`
- golangci-lint `2.12.2`
- GoReleaser `2.16.0`
- glyphrun `v0.1.0-e224a88-dev` or newer for E2E specs

## Verify

```bash
vidtrace version
vidtrace doctor
vidtrace docs agent
```

If `doctor` reports missing OCR language data, install the requested Tesseract language package before extraction.
