#!/bin/sh
# Build a small, deterministic sample artifact bundle for glyphrun flows.
#
# Usage: sample_bundle.sh <bundle-dir> [source-video]
#
# The frame_0002 OCR intentionally includes browser/date noise (https, localhost,
# example.com, 2026, Monday) so investigate flows can assert that suggested code
# searches drop it. Keyword/search flows ignore the extra tokens.
set -eu

dir="$1"
source_video="${2:-/tmp/ticket-bug.mp4}"

mkdir -p "$dir/frames" "$dir/ocr" "$dir/transcript"

cat > "$dir/metadata.json" <<JSON
{
  "schema_version": "1",
  "source_video": "$source_video",
  "duration_seconds": 2,
  "extract_fps": 1,
  "ocr_languages": ["eng"],
  "whisper_language": "en",
  "whisper_model": "small"
}
JSON

cat > "$dir/timeline.json" <<'JSON'
{
  "schema_version": "1",
  "entries": [
    {
      "time_seconds": 0,
      "frame": "frames/frame_0001.png",
      "ocr": {"path": "ocr/frame_0001.txt", "text": "Ticket list"},
      "transcript": [{"start_seconds": 0, "end_seconds": 1, "text": "I am opening the ticket"}]
    },
    {
      "time_seconds": 1,
      "frame": "frames/frame_0002.png",
      "ocr": {"path": "ocr/frame_0002.txt", "text": "Ticket OPG-14010 details https localhost example.com 2026 Monday"},
      "transcript": [{"start_seconds": 1, "end_seconds": 2, "text": "I clicked the ticket and it does not work"}]
    }
  ]
}
JSON

touch "$dir/frames/frame_0001.png" "$dir/frames/frame_0002.png"
printf 'Ticket list\n' > "$dir/ocr/frame_0001.txt"
printf 'Ticket OPG-14010 details https localhost example.com 2026 Monday\n' > "$dir/ocr/frame_0002.txt"
printf 'Ticket list\nTicket OPG-14010 details https localhost example.com 2026 Monday\n' > "$dir/ocr/ocr_all_frames.txt"
