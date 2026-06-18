#!/usr/bin/env bash
# vidtrace — extract frames + OCR + transcript from a bug video.
#
# Usage:
#   ./extract.sh /path/to/video.mp4
#   ./extract.sh                      # defaults to ~/Downloads/bug.mp4
#
# Outputs to: ~/Downloads/<video>_artifacts_<timestamp>/
#
# Requires: ffmpeg, tesseract (eng), whisper

set -euo pipefail

VIDEO="${1:-$HOME/Downloads/bug.mp4}"
VIDEO_BASE="$(basename "$VIDEO" .mp4)"
OUT="$HOME/Downloads/${VIDEO_BASE}_artifacts_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$OUT/frames" "$OUT/ocr" "$OUT/transcript"

echo ">>> Extracting 1 fps frames"
ffmpeg -hide_banner -loglevel error -i "$VIDEO" -vf fps=1 "$OUT/frames/frame_%04d.png"

echo ">>> OCRing frames (tesseract eng)"
for f in "$OUT"/frames/*.png; do
  base="$(basename "$f" .png)"
  tesseract "$f" "$OUT/ocr/$base" -l eng >/dev/null 2>&1
done

echo ">>> Combining OCR"
{
  echo "Video: $VIDEO"
  echo "Generated: $(date)"
  echo
  for t in "$OUT"/ocr/frame_*.txt; do
    echo "===== $(basename "$t") ====="
    cat "$t"
    echo
  done
} > "$OUT/ocr/ocr_all_frames.txt"

echo ">>> Transcribing audio (whisper small, en)"
whisper "$VIDEO" --model small --language en \
  --output_dir "$OUT/transcript" --output_format all

FRAME_COUNT=$(ls "$OUT/frames" | wc -l | tr -d " ")
OCR_COUNT=$(ls "$OUT"/ocr/frame_*.txt | wc -l | tr -d " ")
{
  echo "Output folder: $OUT"
  echo "Frames: $FRAME_COUNT"
  echo "OCR frame txt files: $OCR_COUNT"
  echo "Combined OCR: ocr/ocr_all_frames.txt"
  echo "Transcript files:"
  ls -1 "$OUT/transcript"
} > "$OUT/README.txt"

echo "Done: $OUT"
