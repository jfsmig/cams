// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	_ "go.nanomsg.org/mangos/v3/transport/inproc"

	"github.com/jfsmig/cams/go/utils"
)

var (
	user         = "admin"
	password     = "ollyhgqo"
	upstreamAddr = "127.0.0.1:6000"
)

var (
	httpClient = http.Client{}
)

func main() {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Cams Agent",
		Long:  "LAN agent for OnVif cameras",
		//Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// FIXME(jfs): load an external configuration file or CLI options
			cfg := DefaultConfig()
			ctx, cancel := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)
			defer cancel()

			return runAgent(ctx, cfg)
		},
	}

	if err := cmd.Execute(); err != nil {
		utils.Logger.Fatal().Err(err).Str("action", "aborting").Msg("agent")
	} else {
		utils.Logger.Info().Str("action", "Exiting").Msg("agent")
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

	utils.Logger.Info().Str("action", "starting").Msg("agent")

	utils.GroupRun(ctx,
		func(c context.Context) { upstream.Run(c, lan) },
		func(c context.Context) { lan.Run(c) },
	)

	return nil
}
