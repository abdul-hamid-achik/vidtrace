package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/abdul-hamid-achik/vidtrace/internal/fcheap"
)

func runStash(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printStashHelp(stderr)
		return 2
	}

	sub := args[0]
	rest := args[1:]

	switch sub {
	case "save":
		return runStashSave(rest, stdout, stderr)
	case "list":
		return runStashList(rest, stdout, stderr)
	case "restore":
		return runStashRestore(rest, stdout, stderr)
	case "info":
		return runStashInfo(rest, stdout, stderr)
	case "search":
		return runStashSearch(rest, stdout, stderr)
	case "help", "-h", "--help":
		printStashHelp(stdout)
		return 0
	default:
		_, _ = fmt.Fprintf(stderr, "unknown stash subcommand: %s\n\n", sub)
		printStashHelp(stderr)
		return 2
	}
}

func runStashSave(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("stash save", flag.ContinueOnError)
	fs.SetOutput(stderr)
	name := fs.String("name", "", "display name for the stash")
	tool := fs.String("tool", "vidtrace", "tool tag for the stash")
	jsonOutput := fs.Bool("json", false, "print machine-readable JSON")

	var tags []string
	normalizedArgs, err := normalizeStashArgs(args, map[string]struct{}{"json": {}}, map[string]struct{}{
		"name": {},
		"tool": {},
	}, &tags)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	if err := fs.Parse(normalizedArgs); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "usage: vidtrace stash save [flags] /path/to/bundle")
		return 2
	}

	if !fcheap.Available() {
		return writeStashFailure(stdout, stderr, *jsonOutput, "fcheap is not installed or not on PATH")
	}

	path, err := expandHome(fs.Arg(0))
	if err != nil {
		return writeStashFailure(stdout, stderr, *jsonOutput, fmt.Sprintf("resolve path: %v", err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := fcheap.Save(ctx, path, *name, *tool, tags)
	if err != nil {
		return writeStashFailure(stdout, stderr, *jsonOutput, err.Error())
	}

	if *jsonOutput {
		_ = writeJSON(stdout, map[string]any{
			"ok":   true,
			"id":   result.ID,
			"name": result.Name,
		})
	} else {
		_, _ = fmt.Fprintf(stdout, "Stashed: %s\n", result.ID)
		if result.Name != "" {
			_, _ = fmt.Fprintf(stdout, "Name: %s\n", result.Name)
		}
	}
	return 0
}

func runStashList(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("stash list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	tool := fs.String("tool", "", "filter by tool tag")
	tag := fs.String("tag", "", "filter by tag")
	jsonOutput := fs.Bool("json", false, "print machine-readable JSON")

	normalizedArgs, err := normalizeStashArgs(args, map[string]struct{}{"json": {}}, map[string]struct{}{
		"tool": {},
		"tag":  {},
	}, nil)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	if err := fs.Parse(normalizedArgs); err != nil {
		return 2
	}

	if !fcheap.Available() {
		return writeStashFailure(stdout, stderr, *jsonOutput, "fcheap is not installed or not on PATH")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entries, err := fcheap.List(ctx, *tool, *tag)
	if err != nil {
		return writeStashFailure(stdout, stderr, *jsonOutput, err.Error())
	}

	if *jsonOutput {
		_ = writeJSON(stdout, map[string]any{
			"ok":      true,
			"stashes": entries,
		})
	} else {
		if len(entries) == 0 {
			_, _ = fmt.Fprintln(stdout, "No stashes found.")
			return 0
		}
		for _, e := range entries {
			_, _ = fmt.Fprintf(stdout, "  %s  %s  (%d files)\n", e.ID, e.Name, e.FileCount)
		}
	}
	return 0
}

func runStashRestore(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("stash restore", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String("to", "", "target directory (default: fresh temp dir)")
	jsonOutput := fs.Bool("json", false, "print machine-readable JSON")

	normalizedArgs, err := normalizeStashArgs(args, map[string]struct{}{"json": {}}, map[string]struct{}{
		"to": {},
	}, nil)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	if err := fs.Parse(normalizedArgs); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "usage: vidtrace stash restore [flags] <stash-id>")
		return 2
	}

	if !fcheap.Available() {
		return writeStashFailure(stdout, stderr, *jsonOutput, "fcheap is not installed or not on PATH")
	}

	resolvedTarget := strings.TrimSpace(*target)
	if resolvedTarget != "" {
		resolvedTarget, err = expandHome(resolvedTarget)
		if err != nil {
			return writeStashFailure(stdout, stderr, *jsonOutput, fmt.Sprintf("resolve target: %v", err))
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	path, err := fcheap.Restore(ctx, fs.Arg(0), resolvedTarget)
	if err != nil {
		return writeStashFailure(stdout, stderr, *jsonOutput, err.Error())
	}

	if *jsonOutput {
		_ = writeJSON(stdout, map[string]any{
			"ok":   true,
			"path": path,
		})
	} else {
		_, _ = fmt.Fprintf(stdout, "Restored to: %s\n", path)
	}
	return 0
}

func runStashInfo(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("stash info", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "print machine-readable JSON")

	normalizedArgs, err := normalizeStashArgs(args, map[string]struct{}{"json": {}}, nil, nil)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	if err := fs.Parse(normalizedArgs); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "usage: vidtrace stash info <stash-id>")
		return 2
	}

	if !fcheap.Available() {
		return writeStashFailure(stdout, stderr, *jsonOutput, "fcheap is not installed or not on PATH")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	info, err := fcheap.Info(ctx, fs.Arg(0))
	if err != nil {
		return writeStashFailure(stdout, stderr, *jsonOutput, err.Error())
	}

	if *jsonOutput {
		_ = writeJSON(stdout, info)
	} else {
		_, _ = fmt.Fprintf(stdout, "ID: %s\n", info.ID)
		_, _ = fmt.Fprintf(stdout, "Name: %s\n", info.Name)
		if info.Tool != "" {
			_, _ = fmt.Fprintf(stdout, "Tool: %s\n", info.Tool)
		}
		_, _ = fmt.Fprintf(stdout, "Files: %d\n", info.FileCount)
		if info.TotalSize > 0 {
			_, _ = fmt.Fprintf(stdout, "Size: %d bytes\n", info.TotalSize)
		}
		if len(info.Tags) > 0 {
			_, _ = fmt.Fprintf(stdout, "Tags: %s\n", strings.Join(info.Tags, ", "))
		}
	}
	return 0
}

func runStashSearch(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("stash search", flag.ContinueOnError)
	fs.SetOutput(stderr)
	mode := fs.String("mode", "", "search mode: keyword, semantic, or hybrid")
	limit := fs.Int("limit", 20, "maximum results")
	jsonOutput := fs.Bool("json", false, "print machine-readable JSON")

	normalizedArgs, err := normalizeStashArgs(args, map[string]struct{}{"json": {}}, map[string]struct{}{
		"mode":  {},
		"limit": {},
	}, nil)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	if err := fs.Parse(normalizedArgs); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "usage: vidtrace stash search [flags] <query>")
		return 2
	}

	if !fcheap.Available() {
		return writeStashFailure(stdout, stderr, *jsonOutput, "fcheap is not installed or not on PATH")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := fcheap.Search(ctx, fs.Arg(0), *mode, *limit)
	if err != nil {
		return writeStashFailure(stdout, stderr, *jsonOutput, err.Error())
	}

	if *jsonOutput {
		_ = writeJSON(stdout, result)
	} else {
		if len(result.Matches) == 0 {
			_, _ = fmt.Fprintln(stdout, "No matches found.")
			return 0
		}
		for _, m := range result.Matches {
			file := m.File
			if file == "" {
				file = m.StashID
			}
			_, _ = fmt.Fprintf(stdout, "  %.4f  %s  %s\n", m.Score, file, truncateStashText(m.Text, 80))
		}
	}
	return 0
}

func writeStashFailure(stdout, stderr io.Writer, jsonOutput bool, message string) int {
	if jsonOutput {
		_ = writeJSON(stdout, map[string]any{
			"ok":    false,
			"error": message,
		})
	} else {
		_, _ = fmt.Fprintf(stderr, "stash failed: %s\n", message)
	}
	return 1
}

func truncateStashText(text string, limit int) string {
	text = strings.Join(strings.Fields(text), " ")
	if len(text) <= limit {
		return text
	}
	if limit <= 1 {
		return text[:limit]
	}
	return text[:limit-1] + "..."
}

func printStashHelp(w io.Writer) {
	_, _ = fmt.Fprint(w, `vidtrace stash - manage artifact bundles in the fcheap vault

Usage:
  vidtrace stash <subcommand> [flags]

Subcommands:
  save     Save a bundle or directory to the stash vault (--tag can be repeated)
  list     List stashes, optionally filtered by tool or tag
  restore  Restore a stash to a local directory
  info     Show metadata and file list for a stash
  search   Search across all indexed stashes

Examples:
  vidtrace stash save /path/to/bundle --name "OPG-15070 bug" --tag bug
  vidtrace stash list --tool vidtrace --json
  vidtrace stash restore <stash-id> --to /tmp/restored
  vidtrace stash info <stash-id> --json
  vidtrace stash search "login fails" --json
`)
}

// normalizeStashArgs separates flags from positionals and collects repeated
// --tag flags into the tags slice. It follows the existing normalizeBundleArgs
// pattern but adds multi-value tag support.
func normalizeStashArgs(args []string, boolFlags, valueFlags map[string]struct{}, tags *[]string) ([]string, error) {
	var flags []string
	var positionals []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if len(arg) == 0 || arg[0] != '-' || arg == "-" {
			positionals = append(positionals, arg)
			continue
		}

		name := trimFlagName(arg)

		// Collect --tag values into the tags slice. The tag flags are consumed
		// here and NOT passed through to fs.Parse, since the flag set does not
		// define a -tag flag.
		if name == "tag" && tags != nil {
			if hasInlineValue(arg) {
				if _, after, ok := strings.Cut(arg, "="); ok {
					*tags = append(*tags, after)
				}
			} else {
				if i+1 >= len(args) {
					return nil, fmt.Errorf("missing value for flag %s", arg)
				}
				i++
				*tags = append(*tags, args[i])
			}
			continue
		}

		if _, ok := boolFlags[name]; ok {
			flags = append(flags, arg)
			continue
		}

		if _, ok := valueFlags[name]; ok {
			flags = append(flags, arg)
			if !hasInlineValue(arg) {
				if i+1 >= len(args) {
					return nil, fmt.Errorf("missing value for flag %s", arg)
				}
				i++
				flags = append(flags, args[i])
			}
			continue
		}

		flags = append(flags, arg)
	}
	return append(flags, positionals...), nil
}
