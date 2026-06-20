package cli

import (
	"context"
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

	// Serve normalizes a normal client disconnect to nil; anything returned here
	// is a real error.
	if err := mcpserver.Serve(ctx, version); err != nil {
		_, _ = fmt.Fprintf(stderr, "mcp server error: %v\n", err)
		return 1
	}
	return 0
}
