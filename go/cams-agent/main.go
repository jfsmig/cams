// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	utils2 "github.com/jfsmig/cams/go/utils"
	"github.com/spf13/cobra"
	_ "go.nanomsg.org/mangos/v3/transport/inproc"
)

var (
	user         = "admin"
	password     = "ollyhgqo"
	upstreamAddr = "127.0.0.1:6000"
)

func main() {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Cams Agent",
		Long:  "LAN agent for OnVif cameras",
		//Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := AgentConfig{
				User: "plop",

				DiscoverPatterns: []string{"!lo", "!docker.*", ".*"},
				Upstream: UpstreamConfig{
					Address: "127.0.0.1:6000",
					Timeout: 30,
				},
			}

			ctx := context.Background()
			return runAgent(ctx, cfg)
		},
	}

	if err := cmd.Execute(); err != nil {
		utils2.Logger.Fatal().Err(err).Str("action", "aborting").Msg("agent")
	} else {
		utils2.Logger.Info().Str("action", "Exiting").Msg("agent")
	}
}

func runAgent(ctx context.Context, cfg AgentConfig) error {
	lan := NewLanAgent(cfg)
	upstream := NewUpstreamAgent(cfg)

	// Let the upstream close the upstream for disappeared cameras
	lan.AttachCameraObserver(upstream)
	defer lan.DetachCameraObserver(upstream)

	// Let the lan start/stop the streaming based on the command down the upstream
	upstream.AttachCommandObserver(lan)
	defer upstream.DetachCommandObserver(lan)

	utils2.Logger.Info().Str("action", "starting").Msg("agent")

	utils2.GroupRun(ctx,
		func(c context.Context) { upstream.Run(c, lan) },
		func(c context.Context) { lan.Run(c) },
	)

	return nil
}