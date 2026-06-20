package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/abdul-hamid-achik/vidtrace/internal/mcpserver"
)

func runMCP(args []string, stderr io.Writer, version string) int {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		_, _ = fmt.Fprintln(stderr, "usage: vidtrace mcp")
		return 2
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := mcpserver.Serve(ctx, version); err != nil &&
		!errors.Is(err, context.Canceled) && !errors.Is(err, io.EOF) {
		_, _ = fmt.Fprintf(stderr, "mcp server error: %v\n", err)
		return 1
	}
	return 0
}
