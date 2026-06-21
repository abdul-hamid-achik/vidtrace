# Artifact Schema

The artifact bundle is the product surface that agents and humans consume.

## Bundle Layout

```text
<video-name>_artifacts_<timestamp>/
├── frames/
├── ocr/
├── transcript/
├── metadata.json
├── timeline.json
└── README.txt
```

## `metadata.json`

Target shape:

```json
{
  "schema_version": "1",
  "source_video": "/absolute/path/to/bug.mp4",
  "generated_at": "2026-06-18T17:00:00Z",
  "duration_seconds": 123.45,
  "width": 1920,
  "height": 1080,
  "video_codec": "h264",
  "audio_codec": "aac",
  "frame_rate": 30,
  "extract_fps": 1,
  "ocr_languages": ["eng"],
  "whisper_language": "en",
  "whisper_model": "small"
}
```

## `timeline.json`

Target shape:

```json
{
  "schema_version": "1",
  "entries": [
    {
      "time_seconds": 75,
      "frame": "frames/frame_0075.png",
      "ocr": {
        "path": "ocr/frame_0075.txt",
        "text": "NetworkError"
      },
      "transcript": [
        {
          "start_seconds": 73.2,
          "end_seconds": 77.8,
          "text": "This is broken after submit."
        }
      ]
    }
  ]
}
```

`time_seconds` is calculated from the frame index and extraction FPS:

```text
time_seconds = (frame_index - 1) / extract_fps
```

For example, `frame_0001.png` is `0` seconds and `frame_0002.png` is `0.5` seconds when `extract_fps` is `2`.

Empty OCR text is represented as an empty string. This means OCR ran for the frame but no text was detected or retained.

### Transcript matching

Each frame owns the half-open interval from its own `time_seconds` to the **next frame's actual `time_seconds`** (the last frame owns everything to the end of the recording). A transcript segment is attached to every frame whose interval it overlaps, so a sentence spoken across several frames appears on each of them, and the screen visible while it was spoken is always reachable.

This model tiles the whole recording with no gaps and behaves correctly for fractional frame rates and for sparse extraction (when frames are missing, a frame's interval extends to the next frame that actually exists). Because the interval is half-open, a segment that touches a frame boundary is attached to a single frame, not double-counted. A segment that still overlaps no interval — for example a zero-length segment exactly on a boundary — falls back to the nearest frame by its midpoint, so no transcript is dropped.

Limits: matching is purely temporal; it does not align spoken words to the specific on-screen element they refer to, and at sparse frame rates a frame can own a long interval, so an attached segment may have been spoken several seconds before or after that frame was captured.

## Compatibility

- Add new fields instead of changing field meaning.
- Keep `schema_version` as a string.
- Prefer relative paths inside the bundle.
- Avoid embedding large binary data in JSON.

## Consumers

- `vidtrace studio` reads `metadata.json`, `timeline.json`, OCR text, transcript text, and frame paths.
- `vidtrace analyze` and `vidtrace compare` read bundle text evidence and a ticket file.
- `vidtrace validate` checks required files, parseable JSON, timeline entries, and referenced frame/OCR paths. It also emits soft `warnings` (without failing validation) when `transcript/` is empty despite a declared whisper model, or when the frame count differs from the OCR frame txt count.
- Agents should treat `timeline.json` as the primary map from timestamp to visual and spoken evidence.

## Bundle Directory Uniqueness

The pipeline names bundles `<name>_artifacts_<YYYYMMDD_HHMMSS>`. If a directory with that name already exists (for example two runs in the same second), a numeric suffix is appended (`_2`, `_3`, ...) so runs never silently overwrite each other.
