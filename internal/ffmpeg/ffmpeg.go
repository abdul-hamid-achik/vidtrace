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
