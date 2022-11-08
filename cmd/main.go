// Package main contains a useful demonstration application showing how to
// watch a directory for changes and print the results to stdout.
//
// Usage:
//
//	globwatch [--pattern <pattern>] [--interval <interval>] <directory>
//
// This starts the detection which runs until SIGINT is received which causes
// the app to do a graceful shutdown.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/halimath/globwatch"
)

var (
	pattern  = flag.String("pattern", "**/*", "Pattern of files to watch")
	interval = flag.Duration("interval", time.Second, "Interval to check for changes")
)

func main() {
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "%s: missing directory\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Usage: %s [--pattern <PATTERN>] <DIR>\n", os.Args[0])
		os.Exit(1)
	}

	dir, err := filepath.Abs(flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: unable to create watcher: %s\n", os.Args[0], err)
		os.Exit(2)
	}

	watcher, err := globwatch.New(os.DirFS(dir), *pattern, *interval)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: unable to create watcher: %s\n", os.Args[0], err)
		os.Exit(2)
	}

	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt, syscall.SIGINT)

	if err := watcher.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: unable to start watcher: %s\n", os.Args[0], err)
		os.Exit(3)
	}

	go func() {
		for err := range watcher.ErrorsChan() {
			fmt.Fprintf(os.Stderr, "%s: failed to detect changes: %s\n", os.Args[0], err)
		}
	}()

	go func() {
		for e := range watcher.C() {
			fmt.Printf("%8s %s\n", e.Type, e.Path)
		}
	}()

	<-s

	watcher.Close()
}
