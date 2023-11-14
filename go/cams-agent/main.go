// Copyright (c) 2022-2024 The authors (see the AUTHORS file)
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/jfsmig/cams/go/utils"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

func main() {
	var flagSpeed bool

	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Cams Agent",
		Long:  "LAN agent for OnVif cameras",
		//Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			zerolog.SetGlobalLevel(zerolog.TraceLevel)
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
