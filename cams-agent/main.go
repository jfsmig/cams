// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var (
	user         = "admin"
	password     = "ollyhgqo"
	upstreamAddr = "127.0.0.1:6000"
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

func run(ctx context.Context, cancel context.CancelFunc, cfg AgentConfig) error {
	defer cancel()

	wg := sync.WaitGroup{}
	upstream := NewUpstreamAgent(ctx, cancel, &wg)
	lan := NewLanAgent(ctx, cancel, &wg)

	lan.Configure(cfg)

	wg.Add(2)
	go upstream.Run(upstreamAddr)
	go lan.Run()
	wg.Wait()

	return nil
}

func main() {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Cams Agent",
		Long:  "LAN agent for OnVif cameras",
		//Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			cfg := AgentConfig{
				DiscoverPatterns: []string{"*"},
			}

			return run(ctx, cancel, cfg)
		},
	}

	if err := cmd.Execute(); err != nil {
		Logger.Fatal().Err(err).Msg("Aborting")
	} else {
		Logger.Info().Msg("Exiting")
	}
}
