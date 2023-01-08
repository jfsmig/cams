// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	_ "go.nanomsg.org/mangos/v3/transport/inproc"

	"github.com/jfsmig/cams/go/utils"
)

func main() {
	var flagSpeed bool

	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Cams Agent",
		Long:  "LAN agent for OnVif cameras",
		//Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// FIXME(jfs): load an external configuration file or CLI options
			cfg := DefaultConfig()
			if flagSpeed {
				cfg.RegisterPeriod = 1
				cfg.ScanPeriod = 0
				cfg.CheckPeriod = 1
			}
			ctx, cancel := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)
			defer cancel()

			return runAgent(ctx, cfg)
		},
	}

	cmd.Flags().BoolVarP(&flagSpeed, "speed", "s", false, "TEST with fast loops")

	if err := cmd.Execute(); err != nil {
		utils.Logger.Fatal().Err(err).Str("action", "aborting").Msg("agent")
	} else {
		utils.Logger.Info().Str("action", "Exiting").Msg("agent")
	}
}

func runAgent(ctx context.Context, cfg AgentConfig) error {
	lan := NewLanAgent(cfg)
	upstream := NewUpstreamAgent(cfg)

	utils.Logger.Info().Str("action", "starting").Msg("agent")

	utils.GroupRun(ctx,
		func(c context.Context) { upstream.Run(c, lan) },
		func(c context.Context) { lan.Run(c) },
	)

	return nil
}
