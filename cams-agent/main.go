// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"github.com/jfsmig/cams/utils"
	"github.com/spf13/cobra"
	"sync"
)

var (
	user         = "admin"
	password     = "ollyhgqo"
	upstreamAddr = "127.0.0.1:6000"
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
		utils.Logger.Fatal().Err(err).Msg("Aborting")
	} else {
		utils.Logger.Info().Msg("Exiting")
	}
}
