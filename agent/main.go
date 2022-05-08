package main

import (
	"github.com/rs/zerolog"

	"context"
	"log"
	"net"
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

func main() {
	agent := NewLanAgent()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg := sync.WaitGroup{}

	itfs, err := net.Interfaces()
	if err != nil {
		log.Panicln(err)
	}
	for _, itf := range itfs {
		if itf.Name != "lo" {
			wg.Add(1)
			agent.RegisterInterface(itf.Name)
		}
	}

	go agent.RunLoop(ctx, &wg)
	wg.Wait()
}
