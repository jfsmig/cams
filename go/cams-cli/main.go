// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"github.com/jfsmig/cams/go/utils"
	"github.com/juju/errors"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
)

func main() {
	cmd := &cobra.Command{
		Use:   "cams",
		Short: "Cams command Line Interface",
		Long:  "CLI Client for Cams services",
		//Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("Missing sub-command")
		},
	}

	play := &cobra.Command{
		Use:   "play",
		Short: "Play a stream",
		Long:  "Contact the Cams Hub and download a stream given its User/Stream ID",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)
			defer cancel()
			return play(ctx, "127.0.0.1:6000", args[0], args[1])
		},
	}

	discover := &cobra.Command{
		Use:   "discover",
		Short: "Discover the local cameras",
		Long:  "Discover the local caneras the way the Cams Agent does",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)
			defer cancel()
			return discover(ctx)
		},
	}

	details := &cobra.Command{
		Use:   "detail",
		Short: "Dump the configuration of the given camera",
		Long:  "Dump the configuration of the given camera, playing all the possible OnVif calls to explicitly check which are supported",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)
			defer cancel()
			return details(ctx, args[0])
		},
	}

	cmd.AddCommand(play, discover, details)

	if err := cmd.Execute(); err != nil {
		utils.Logger.Fatal().Err(err).Msg("Aborting")
	} else {
		utils.Logger.Info().Msg("Exiting")
	}
}
