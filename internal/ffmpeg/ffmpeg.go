package ffmpeg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type Metadata struct {
	DurationSeconds float64
	Width           int
	Height          int
	VideoCodec      string
	AudioCodec      string
	FrameRate       float64
}

type probeOutput struct {
	Streams []probeStream `json:"streams"`
	Format  probeFormat   `json:"format"`
}

type probeStream struct {
	CodecName    string `json:"codec_name"`
	CodecType    string `json:"codec_type"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	AvgFrameRate string `json:"avg_frame_rate"`
	RFrameRate   string `json:"r_frame_rate"`
}

type probeFormat struct {
	Duration string `json:"duration"`
}

func Probe(ctx context.Context, videoPath string) (Metadata, error) {
	output, err := run(ctx, "ffprobe",
		"-v", "error",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		videoPath,
	)
	if err != nil {
		return Metadata{}, err
	}

	var probe probeOutput
	if err := json.Unmarshal(output, &probe); err != nil {
		return Metadata{}, fmt.Errorf("parse ffprobe json: %w", err)
	}

	var metadata Metadata
	metadata.DurationSeconds, _ = strconv.ParseFloat(probe.Format.Duration, 64)

	for _, stream := range probe.Streams {
		switch stream.CodecType {
		case "video":
			if metadata.VideoCodec == "" {
				metadata.VideoCodec = stream.CodecName
				metadata.Width = stream.Width
				metadata.Height = stream.Height
				metadata.FrameRate = parseRate(firstNonEmpty(stream.AvgFrameRate, stream.RFrameRate))
			}
		case "audio":
			if metadata.AudioCodec == "" {
				metadata.AudioCodec = stream.CodecName
			}
		}
	}

	return metadata, nil
}

func ExtractFrames(ctx context.Context, videoPath string, fps float64, outputPattern string) error {
	_, err := run(ctx, "ffmpeg",
		"-hide_banner",
		"-loglevel", "error",
		"-i", videoPath,
		"-vf", "fps="+formatFloat(fps),
		outputPattern,
	)
	return err
}

// CutClip extracts a sub-clip from videoPath starting at startSec and ending at
// endSec, writing to outputPath. When reencode is false (the default) it uses
// stream copy (-c copy) for speed; when true it re-encodes with libx264/aac,
// which is slower but works when stream copy produces seeking artifacts.
func CutClip(ctx context.Context, videoPath string, startSec, endSec float64, outputPath string, reencode bool) error {
	args := []string{
		"-hide_banner",
		"-loglevel", "error",
		"-ss", formatFloat(startSec),
		"-to", formatFloat(endSec),
		"-i", videoPath,
	}
	if reencode {
		args = append(args, "-c:v", "libx264", "-c:a", "aac")
	} else {
		args = append(args, "-c", "copy")
	}
	args = append(args, outputPath)
	_, err := run(ctx, "ffmpeg", args...)
	return err
}

// MakeGIF creates an animated GIF from a time range in the video. fps controls
// the GIF frame rate (10 is a good default for UI bugs); width controls the
// output pixel width (height auto-scales to preserve aspect ratio). It uses
// a two-pass palette approach for high quality without banding.
func MakeGIF(ctx context.Context, videoPath string, startSec, endSec float64, outputPath string, fps, width int) error {
	filter := fmt.Sprintf("fps=%d,scale=%d:-1:flags=lanczos,split[s0][s1];[s0]palettegen[p];[s1][p]paletteuse", fps, width)
	_, err := run(ctx, "ffmpeg",
		"-hide_banner",
		"-loglevel", "error",
		"-ss", formatFloat(startSec),
		"-to", formatFloat(endSec),
		"-i", videoPath,
		"-vf", filter,
		"-loop", "0",
		outputPath,
	)
	return err
}

// ConcatClips concatenates the clips listed in listFilePath into outputPath.
// listFilePath must be a text file with one "file 'path'" line per clip. Uses
// the concat demuxer with stream copy, which is fast and avoids re-encoding
// when all clips share the same codec parameters (e.g. cut from the same source).
func ConcatClips(ctx context.Context, listFilePath, outputPath string) error {
	_, err := run(ctx, "ffmpeg",
		"-hide_banner",
		"-loglevel", "error",
		"-f", "concat",
		"-safe", "0",
		"-i", listFilePath,
		"-c", "copy",
		outputPath,
	)
	return err
}

func run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return stdout.Bytes(), fmt.Errorf("%s failed: %w: %s", name, err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

func parseRate(rate string) float64 {
	parts := strings.Split(rate, "/")
	if len(parts) == 2 {
		numerator, nerr := strconv.ParseFloat(parts[0], 64)
		denominator, derr := strconv.ParseFloat(parts[1], 64)
		if nerr == nil && derr == nil && denominator != 0 {
			return numerator / denominator
		}
	}
	value, _ := strconv.ParseFloat(rate, 64)
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
