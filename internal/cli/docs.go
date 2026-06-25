package cli

import (
	"fmt"
	"io"
	"strings"
)

func runDocs(args []string, stdout, stderr io.Writer) int {
	topic := "overview"
	if len(args) > 1 {
		_, _ = fmt.Fprintln(stderr, "usage: vidtrace docs [overview|agent|commands|artifacts|studio]")
		return 2
	}
	if len(args) == 1 {
		topic = strings.ToLower(args[0])
	}

	switch topic {
	case "", "overview":
		printOverviewDocs(stdout)
	case "agent":
		printAgentDocs(stdout)
	case "commands":
		printCommandDocs(stdout)
	case "artifacts":
		printArtifactDocs(stdout)
	case "studio":
		printStudioDocs(stdout)
	default:
		_, _ = fmt.Fprintf(stderr, "unknown docs topic: %s\n", topic)
		_, _ = fmt.Fprintln(stderr, "usage: vidtrace docs [overview|agent|commands|artifacts|studio]")
		return 2
	}
	return 0
}

func printOverviewDocs(w io.Writer) {
	_, _ = fmt.Fprint(w, `vidtrace product docs

What it does:
  vidtrace turns bug screen recordings into evidence bundles that humans and coding agents can inspect.

Primary output:
  - frames/                 extracted screenshots
  - ocr/                    Tesseract text per frame plus combined OCR
  - transcript/             Whisper transcript files
  - metadata.json           source video and run metadata
  - timeline.json           timestamped OCR plus transcript evidence

Common workflows:
  - Run "vidtrace doctor" before extraction.
  - Run "vidtrace extract VIDEO --json" for automation.
  - Run "vidtrace validate BUNDLE --json" before trusting a generated bundle.
  - Run "vidtrace index BUNDLE --db evidence.veclite --json" to make timeline evidence searchable.
  - Run "vidtrace search evidence.veclite QUERY --json" to find timestamped evidence.
  - Run "vidtrace investigate BUNDLE --query QUERY --codebase REPO --json" for code-search handoff.
  - Run "vidtrace investigate BUNDLE --query QUERY --codebase REPO --connect --json" for real code matches via fcheap.
  - Run "vidtrace investigate BUNDLE --query QUERY --codebase REPO --connect --codemap --json" for symbol resolution and blast radius via codemap.
  - Run "vidtrace stash save BUNDLE --json" to save a bundle to the fcheap vault.
  - Run "vidtrace clip cut VIDEO --label ISSUE=START-END --json" to cut clips from a video.
  - Run "vidtrace clip gif VIDEO --label ISSUE=START-END --json" to make GIFs from a video.
  - Run "vidtrace clip stitch clip1.mp4 clip2.mp4 --json" to join clips into one video.
  - Run "vidtrace compare BUNDLE --ticket TICKET --json" to compare a ticket with evidence.
  - Run "vidtrace docs agent" when an agent needs the operating contract.
  - Run "vidtrace studio BUNDLE" to inspect timeline, OCR, transcript, and frame paths.
  - Read docs/SITE.md for the VitePress documentation site and Vercel deployment.

More topics:
  vidtrace docs agent
  vidtrace docs commands
  vidtrace docs artifacts
  vidtrace docs studio
`)
}

func printAgentDocs(w io.Writer) {
	_, _ = fmt.Fprint(w, `vidtrace agent guide

Goal:
  Convert a bug video into files an agent can cite while investigating a ticket.

Recommended flow:
  1. Run "vidtrace doctor -json" and check that ok is true.
  2. Run "vidtrace extract VIDEO --json --out /tmp/vidtrace --name TICKET_ID".
  3. Parse stdout as JSON and read output_dir.
  4. Run "vidtrace validate output_dir --json" and check that ok is true.
  5. Inspect output_dir/metadata.json and output_dir/timeline.json first.
  6. Optionally run "vidtrace index output_dir --db /tmp/vidtrace-evidence.veclite --json".
  7. Optionally run "vidtrace search /tmp/vidtrace-evidence.veclite QUERY --json".
  8. Optionally run "vidtrace investigate output_dir --query QUERY --codebase REPO --json".
  9. Optionally run "vidtrace investigate output_dir --query QUERY --codebase REPO --connect --json" for real code matches.
  10. Optionally run "vidtrace investigate output_dir --query QUERY --codebase REPO --connect --codemap --json" for symbol resolution, callers, and blast radius.
  11. Optionally run "vidtrace stash save output_dir --json" to stash the bundle to fcheap.
  12. Optionally run "vidtrace investigate --stash STASH_ID --query QUERY --json" to investigate a stashed bundle.
  13. Optionally run "vidtrace clip cut VIDEO --label ISSUE=START-END --stash --json" to cut and stash per-issue clips.
  14. Optionally run "vidtrace clip gif VIDEO --label ISSUE=START-END --json" to make GIFs for tickets.
  15. Optionally run "vidtrace clip stitch clip1.mp4 clip2.mp4 --json" to join clips into a summary video.
  16. Use ocr/ocr_all_frames.txt for broad UI text search.
  17. Use transcript/*.json for spoken context.
  18. Open selected frames/frame_*.png only when text evidence is ambiguous.
  19. Run "vidtrace compare output_dir --ticket ticket.md --json" for a first-pass mismatch check.

Ticket comparison:
  - State whether the ticket and video appear to match, mismatch, or are inconclusive.
  - Cite timeline timestamps and relative file paths.
  - Call out when the video shows a different screen, flow, error, account, environment, or timestamp than the ticket describes.

Rules:
  - Treat --json stdout as the automation contract.
  - Do not rely on human progress text.
  - Do not move or rewrite generated artifacts.
  - Do not commit source videos or artifact bundles.
`)
}

func printCommandDocs(w io.Writer) {
	_, _ = fmt.Fprint(w, `vidtrace command docs

Commands:
  vidtrace analyze BUNDLE --ticket TICKET
      Write a Markdown report that compares ticket text with extracted evidence.

  vidtrace compare BUNDLE --ticket TICKET [--json]
      Print match, mismatch, or inconclusive based on OCR/transcript evidence.

  vidtrace doctor [-json]
      Check ffmpeg, ffprobe, tesseract, whisper, OCR languages, and cached Whisper models.

  vidtrace extract VIDEO [flags]
      Generate frames, OCR, transcript, metadata, and timeline artifacts.

  vidtrace index BUNDLE --db DB [--json]
      Index timeline evidence into a local VecLite database.

  vidtrace investigate BUNDLE --query TEXT [--codebase REPO] [--connect] [--codemap] [--stash ID] [--db DB] [--json]
      Return video evidence, code-search queries, vecgrep command suggestions, and real code matches (--connect).
      With --codemap, resolve code matches to enclosing symbols, list callers, and compute blast radius.

  vidtrace clip cut|gif|stitch ...
      Cut clips, make GIFs, or stitch videos from timestamp ranges.

  vidtrace stash save|list|restore|info|search ...
      Manage artifact bundles in the fcheap vault (save, list, restore, info, search).

  vidtrace search DB QUERY [--limit N] [--json]
      Search indexed evidence and return timestamped OCR/transcript results.

  vidtrace docs [overview|agent|commands|artifacts|studio]
      Print built-in usage docs for humans and agents.

  vidtrace studio [BUNDLE]
      Open the artifact inspection studio. With a bundle path, browse timeline, metadata, OCR, transcript, and frame paths.

  vidtrace validate BUNDLE [--json]
      Check that an artifact bundle has required files, parseable JSON, timeline entries, and referenced frame/OCR paths.

  vidtrace version
      Print the CLI version.

Important extract flags:
  --json              emit machine-readable summary only
  --out DIR           parent output directory
  --name NAME         artifact bundle name prefix
  --fps N             frame extraction rate
  --ocr-lang LANG     Tesseract language list, for example eng or eng+spa
  --whisper-lang LANG Whisper language
  --model NAME        Whisper model

Analyze and compare flags:
  --ticket PATH       ticket markdown or text file
  --json              emit machine-readable compare result

Validate flags:
  --json              emit machine-readable validation report

Evidence search flags:
  --db PATH           evidence database path for "vidtrace index"
  --limit N           maximum "vidtrace search" results, default 10
  --json              emit machine-readable index/search output

Investigate flags:
  --query TEXT        required bug or evidence query
  --codebase PATH     optional codebase path for vecgrep command suggestions
  --db PATH           optional reusable evidence database path
  --limit N           maximum evidence results, default 5
  --connect           run fcheap connect for real code matches (requires --codebase)
  --stash ID          fcheap stash ID to restore and investigate instead of a local bundle
  --connect-mode      vecgrep search mode: semantic, keyword, or hybrid
  --connect-limit N   maximum code matches from --connect, default 10
  --codemap           run codemap expansion after --connect to resolve symbols, callers, and blast radius
  --codemap-depth N   max hops for codemap blast radius, default 3
  --codemap-annotate  pin vidtrace evidence findings to resolved codemap symbols
  --json              emit machine-readable investigation handoff

Stash subcommands:
  vidtrace stash save BUNDLE [--name NAME] [--tag TAG] [--json]
  vidtrace stash list [--tool TOOL] [--tag TAG] [--json]
  vidtrace stash restore ID [--to DIR] [--json]
  vidtrace stash info ID [--json]
  vidtrace stash search QUERY [--mode MODE] [--limit N] [--json]

Clip subcommands:
  vidtrace clip cut VIDEO [--range RANGE] [--label LABEL=RANGE] [--out DIR] [--name NAME] [--reencode] [--stash] [--tag TAG] [--json]
  vidtrace clip gif VIDEO [--range RANGE] [--label LABEL=RANGE] [--out DIR] [--name NAME] [--fps N] [--width N] [--stash] [--tag TAG] [--json]
  vidtrace clip stitch CLIP1 CLIP2 [...] [--out DIR] [--name NAME] [--json]

Timestamp formats:
  SS          seconds (e.g. 45)
  MM:SS       minutes and seconds (e.g. 3:40)
  HH:MM:SS    hours, minutes, seconds (e.g. 1:23:45)

Range format:
  START-END   e.g. "0:18-3:40" or "14:50-16:14"

Label format:
  LABEL=START-END  e.g. "issue1-blank-row=0:18-3:40"
`)
}

func printArtifactDocs(w io.Writer) {
	_, _ = fmt.Fprint(w, `vidtrace artifact docs

Bundle layout:
  <name>_artifacts_<timestamp>/
    frames/
    ocr/
    transcript/
    metadata.json
    timeline.json
    README.txt

Agent priority:
  1. metadata.json
  2. timeline.json
  3. ocr/ocr_all_frames.txt
  4. transcript/*.json
  5. selected frames/frame_*.png

Contracts:
  - schema_version is a string.
  - JSON paths are relative to the bundle when possible.
  - timeline entries connect time_seconds, frame, OCR text, and transcript segments.
  - Empty OCR is valid evidence that no text was detected for that frame.
  - Run "vidtrace validate BUNDLE --json" to check required files and referenced paths.
`)
}

func printStudioDocs(w io.Writer) {
	_, _ = fmt.Fprint(w, `vidtrace studio docs

Purpose:
  Studio opens an artifact bundle in a terminal review interface for humans.

Open a bundle:
  vidtrace studio /path/to/bug_artifacts_YYYYMMDD_HHMMSS

Keys:
  up/down or k/j      move through timeline entries
  m                   toggle bundle metadata/details
  o                   open the selected frame when possible
  r                   reveal the selected frame in Finder on macOS
  c                   copy a concise evidence summary when clipboard tooling is available
  q, esc, or ctrl+c   exit

Shows:
  - compact status header and action status line
  - bundle source video, duration, extraction FPS, OCR languages, and Whisper model
  - timeline entry count
  - selected timestamp
  - selected frame path
  - OCR text for the selected frame
  - transcript segments that overlap the selected frame time

Limits:
  - Open, reveal, and copy actions depend on platform tools and show a short status message when unavailable.
  - Extraction still runs through "vidtrace extract".
`)
}
