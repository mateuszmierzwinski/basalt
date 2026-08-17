// Command basalt is an interactive picker for the monorepo's Bazel targets.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mateuszmierzwinski/basalt/internal/cli"
)

var (
	noCache   = flag.Bool("nocache", false, "ignore the target cache and re-index from bazel")
	indexOnly = flag.Bool("index", false, "print what was indexed and exit, without starting the UI")
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Interactive picker for the monorepo's Bazel targets.\n")
		fmt.Fprintf(os.Stderr, "Requires $MONOREPO to be set to the absolute path of the checkout.\n\n")
		fmt.Fprintf(os.Stderr, "Keys:\n")
		fmt.Fprintf(os.Stderr, "  type          filter targets        Tab/Shift+Tab  switch panel\n")
		fmt.Fprintf(os.Stderr, "  Up/Down       move selection        PgUp/PgDn      move a page\n")
		fmt.Fprintf(os.Stderr, "  Enter         choose / run          Esc            back out, then quit\n")
		fmt.Fprintf(os.Stderr, "  Ctrl+R        literal/regex         Ctrl+U         clear the query\n")
		fmt.Fprintf(os.Stderr, "  F5            re-index              Ctrl+C         quit\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGHUP)
	defer stop()

	os.Exit(newApp().Run(ctx, cli.Options{NoCache: *noCache, IndexOnly: *indexOnly}))
}
