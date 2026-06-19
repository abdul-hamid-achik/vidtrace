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

## Compatibility

- Add new fields instead of changing field meaning.
- Keep `schema_version` as a string.
- Prefer relative paths inside the bundle.
- Avoid embedding large binary data in JSON.

## Consumers

- `vidtrace studio` reads `metadata.json`, `timeline.json`, OCR text, transcript text, and frame paths.
- `vidtrace analyze` and `vidtrace compare` read bundle text evidence and a ticket file.
- `vidtrace validate` checks required files, parseable JSON, timeline entries, and referenced frame/OCR paths.
- Agents should treat `timeline.json` as the primary map from timestamp to visual and spoken evidence.
