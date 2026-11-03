//go:build !(js && wasm)

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// The first Ctrl+C (or SIGTERM) stops the computation, which still
	// prints its best schedule; a second one kills the program.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ctx.Done()
		stop()
	}()

	if err := run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		switch {
		case errors.Is(err, errAlerts): // reported with the schedule
			os.Exit(2)
		case !errors.Is(err, flag.ErrHelp):
			fmt.Fprintln(os.Stderr, "error:", err)
		}

		os.Exit(1)
	}
}
