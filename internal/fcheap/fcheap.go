// Package fcheap wraps the fcheap CLI so vidtrace can stash, restore, search,
// and connect artifact bundles to codebases. It mirrors how internal/ffmpeg,
// internal/tesseract, and internal/whisper wrap external CLI tools.
package fcheap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// StashEntry is a single stash returned by fcheap list --json.
type StashEntry struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Tool      string   `json:"tool,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	FileCount int      `json:"file_count,omitempty"`
	TotalSize int64    `json:"total_size,omitempty"`
	CreatedAt string   `json:"created_at,omitempty"`
}

// SaveResult is the JSON returned by fcheap save --json.
type SaveResult struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// FileInfo describes a single file inside a stash.
type FileInfo struct {
	Path string `json:"path"`
	Size int64  `json:"size,omitempty"`
}

// StashInfo is the JSON returned by fcheap info --json.
type StashInfo struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Tool      string     `json:"tool,omitempty"`
	Tags      []string   `json:"tags,omitempty"`
	Files     []FileInfo `json:"files,omitempty"`
	FileCount int        `json:"file_count,omitempty"`
	TotalSize int64      `json:"total_size,omitempty"`
	CreatedAt string     `json:"created_at,omitempty"`
}

// SearchMatch is a single search result from fcheap search.
type SearchMatch struct {
	StashID string  `json:"stash_id,omitempty"`
	Score   float64 `json:"score,omitempty"`
	Text    string  `json:"text"`
	File    string  `json:"file,omitempty"`
	Source  string  `json:"source,omitempty"`
}

// SearchResult is the JSON returned by fcheap search --json.
type SearchResult struct {
	Query   string        `json:"query,omitempty"`
	Mode    string        `json:"mode,omitempty"`
	Matches []SearchMatch `json:"matches,omitempty"`
}

// CodeMatch is a single code match from fcheap connect.
type CodeMatch struct {
	StashID string  `json:"stash_id,omitempty"`
	Score   float64 `json:"score,omitempty"`
	Text    string  `json:"text"`
	File    string  `json:"file"`
	Source  string  `json:"source,omitempty"`
}

// ConnectResult is the JSON returned by fcheap connect --json.
type ConnectResult struct {
	StashID  string      `json:"stash_id"`
	Codebase string      `json:"codebase"`
	Query    string      `json:"query,omitempty"`
	Matches  []CodeMatch `json:"matches,omitempty"`
}

// ConnectOptions controls the fcheap connect invocation.
type ConnectOptions struct {
	StashID     string
	CodebaseDir string
	Query       string
	Mode        string
	Limit       int
	Index       bool
}

// Available reports whether the fcheap binary is on PATH.
func Available() bool {
	_, err := exec.LookPath("fcheap")
	return err == nil
}

// Save stashes a file or directory into the fcheap vault.
func Save(ctx context.Context, path, name, tool string, tags []string) (SaveResult, error) {
	args := []string{"save", path, "--json"}
	if name != "" {
		args = append(args, "--name", name)
	}
	if tool != "" {
		args = append(args, "--tool", tool)
	}
	for _, tag := range tags {
		args = append(args, "--tag", tag)
	}

	output, err := run(ctx, args...)
	if err != nil {
		return SaveResult{}, err
	}
	return decodeJSON[SaveResult](output, "save")
}

// List returns stashes in the vault, optionally filtered by tool or tag.
func List(ctx context.Context, tool, tag string) ([]StashEntry, error) {
	args := []string{"list", "--json"}
	if tool != "" {
		args = append(args, "--tool", tool)
	}
	if tag != "" {
		args = append(args, "--tag", tag)
	}

	output, err := run(ctx, args...)
	if err != nil {
		return nil, err
	}
	return decodeJSON[[]StashEntry](output, "list")
}

// Info returns metadata and file list for a specific stash.
func Info(ctx context.Context, stashID string) (StashInfo, error) {
	output, err := run(ctx, "info", stashID, "--json")
	if err != nil {
		return StashInfo{}, err
	}
	return decodeJSON[StashInfo](output, "info")
}

// Restore extracts a stash to a target directory. If target is empty, fcheap
// picks a fresh temp directory and returns it in the result.
func Restore(ctx context.Context, stashID, target string) (string, error) {
	args := []string{"restore", stashID, "--json"}
	if target != "" {
		args = append(args, "--to", target)
	}

	output, err := run(ctx, args...)
	if err != nil {
		return "", err
	}

	var result struct {
		Target string `json:"target"`
		Path   string `json:"path"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return "", fmt.Errorf("parse fcheap restore json: %w", err)
	}
	if result.Target != "" {
		return result.Target, nil
	}
	return result.Path, nil
}

// Search searches across all indexed stashes.
func Search(ctx context.Context, query, mode string, limit int) (SearchResult, error) {
	args := []string{"search", query, "--json"}
	if mode != "" {
		args = append(args, "--mode", mode)
	}
	if limit > 0 {
		args = append(args, "--limit", fmt.Sprintf("%d", limit))
	}

	output, err := run(ctx, args...)
	if err != nil {
		return SearchResult{}, err
	}
	return decodeJSON[SearchResult](output, "search")
}

// Connect runs vecgrep over a codebase using the stashed artifact's text.
func Connect(ctx context.Context, opts ConnectOptions) (ConnectResult, error) {
	args := []string{"connect", opts.StashID, opts.CodebaseDir, "--json"}
	if opts.Query != "" {
		args = append(args, "--query", opts.Query)
	}
	if opts.Mode != "" {
		args = append(args, "--mode", opts.Mode)
	}
	if opts.Limit > 0 {
		args = append(args, "--limit", fmt.Sprintf("%d", opts.Limit))
	}
	if opts.Index {
		args = append(args, "--index")
	}

	output, err := run(ctx, args...)
	if err != nil {
		return ConnectResult{}, err
	}
	return decodeJSON[ConnectResult](output, "connect")
}

// run executes the fcheap CLI and returns stdout. It wraps stderr in the error
// message on failure, matching the pattern in internal/ffmpeg and
// internal/tesseract.
func run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "fcheap", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if msg != "" {
			return nil, fmt.Errorf("fcheap %s: %w: %s", args[0], err, msg)
		}
		return nil, fmt.Errorf("fcheap %s: %w", args[0], err)
	}
	return stdout.Bytes(), nil
}

func decodeJSON[T any](output []byte, command string) (T, error) {
	var result T
	if len(output) == 0 {
		return result, fmt.Errorf("fcheap %s: empty output", command)
	}
	trimmed := bytes.TrimSpace(output)
	if err := json.Unmarshal(trimmed, &result); err != nil {
		return result, fmt.Errorf("parse fcheap %s json: %w", command, err)
	}
	return result, nil
}
