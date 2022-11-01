// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"github.com/jfsmig/cams/utils"
	"github.com/spf13/cobra"
	_ "go.nanomsg.org/mangos/v3/transport/inproc"
)

var (
	user         = "admin"
	password     = "ollyhgqo"
	upstreamAddr = "127.0.0.1:6000"
)

const (
	urlSouth = "inproc://s"
	urlNorth = "inproc://n"
)

func runAgent(ctx context.Context, cfg AgentConfig) error {
	lan := NewLanAgent(cfg)
	upstream := NewUpstreamAgent(cfg)

	lan.AttachObserver(upstream)

	utils.SwarmRun(ctx,
		func(c context.Context) { upstream.Run(c, lan) },
		func(c context.Context) { lan.Run(c) },
	)

	lan.DetachObserver(upstream)
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
			return runAgent(context.Background(), cfg)
		},
	}

	if err := cmd.Execute(); err != nil {
		utils.Logger.Fatal().Err(err).Msg("Aborting")
	} else {
		utils.Logger.Info().Msg("Exiting")
	}
}
