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

func swarmRun(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, cb func(ctx2 context.Context)) {
	defer cancel()
	defer wg.Done()
	cb(ctx)
}

func swarm(ctx0 context.Context, cbs ...func(context.Context)) {
	ctx, cancel := context.WithCancel(ctx0)
	defer cancel()
	wg := sync.WaitGroup{}

	wg.Add(len(cbs))
	for _, cb := range cbs {
		go swarmRun(ctx, cancel, &wg, cb)
	}
	wg.Done()
}

func run(ctx context.Context, cfg AgentConfig) error {
	lan := NewLanAgent()
	lan.Configure(cfg)

	swarm(ctx,
		func(c context.Context) { RunUpstreamAgent(c, upstreamAddr) },
		func(c context.Context) { lan.Run(c) })
	return nil
}

func main() {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Cams Agent",
		Long:  "LAN agent for OnVif cameras",
		//Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := AgentConfig{
				DiscoverPatterns: []string{"!lo", "!docker.*", ".*"},
			}
			return run(context.Background(), cfg)
		},
	}

	if err := cmd.Execute(); err != nil {
		utils.Logger.Fatal().Err(err).Msg("Aborting")
	} else {
		utils.Logger.Info().Msg("Exiting")
	}
}
