// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"github.com/rs/zerolog"
	"os"
	"sync"
	"time"
)

var (
	user     = "admin"
	password = "ollyhgqo"
)

var (
	// LoggerContext is the builder of a zerolog.Logger that is exposed to the application so that
	// options at the CLI might alter the formatting and the output of the logs.
	LoggerContext = zerolog.
			New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
			With().Timestamp()

	// Logger is a zerolog logger, that can be safely used from any part of the application.
	// It gathers the format and the output.
	Logger = LoggerContext.Logger()
)

func run(ctx context.Context, cancel context.CancelFunc) {
	defer cancel()

	wg := sync.WaitGroup{}
	upstream := NewUpstreamAgent(ctx, cancel, &wg)
	agent := NewLanAgent(ctx, cancel, &wg)
	wg.Add(2)
	go upstream.Run()
	go agent.Run()
	wg.Wait()
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	run(ctx, cancel)
}
