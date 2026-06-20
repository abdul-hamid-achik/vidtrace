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

The macOS binary is signed with a Developer ID certificate and notarized by Apple, so it runs without a Gatekeeper warning.

Install common runtime dependencies on macOS:

```bash
brew install ffmpeg tesseract
pipx install openai-whisper
```

Whisper downloads its model on first use. `vidtrace` defaults to the `small` model.

## Linux packages

Tagged releases publish `.deb` and `.rpm` packages (amd64 and arm64) as GitHub release assets. Download the one for your distribution and architecture, then install it:

```bash
# Debian / Ubuntu
sudo dpkg -i vidtrace_<version>_linux_amd64.deb

# Fedora / RHEL
sudo rpm -i vidtrace_<version>_linux_amd64.rpm
```

The package installs `vidtrace` to `/usr/bin`. It does not pull the media tools as hard dependencies (their package names vary and Whisper is installed via pip), so install them separately and confirm with `vidtrace doctor`:

```bash
# Debian / Ubuntu
sudo apt-get install ffmpeg tesseract-ocr
pipx install openai-whisper
```

A `.tar.gz` of just the binary is also attached to every release for manual installation.

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

## OCR language data

`vidtrace extract` defaults to English (`--ocr-lang eng`). To OCR other languages, install the matching Tesseract language data and pass them with `+`, for example `--ocr-lang eng+spa`. Extraction fails fast and names any missing packs before doing work, so install them first:

```bash
# macOS (Homebrew bundles many languages with tesseract; tesseract-lang adds the rest)
brew install tesseract-lang

# Debian/Ubuntu (one package per language, e.g. Spanish)
sudo apt-get install tesseract-ocr-spa
```

List what is installed with `tesseract --list-langs`, or run `vidtrace doctor`, which reports the available Tesseract languages.
