// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"github.com/jfsmig/cams/go/utils"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)
	defer cancel()

	cmd := &cobra.Command{
		Use:   "cams",
		Short: "Cams command Line Interface",
		Long:  "CLI Client for Cams services",
	}

	cmdHub := &cobra.Command{
		Use:   "hub",
		Short: "Commands targetting the hub",
	}

	cmdHubPlay := &cobra.Command{
		Use:   "play",
		Short: "Play a stream",
		Long:  "Contact the Cams Hub and download a stream given its User/Stream ID",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return hubPlay(ctx, "127.0.0.1:6000", args[0], args[1])
		},
	}

	cmdCam := &cobra.Command{
		Use:   "cam",
		Short: "Commands targetting a camera",
	}

	cmdCamPlay := &cobra.Command{
		Use:   "play",
		Short: "play",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return camPlay(ctx, args[0])
		},
	}

	cmdCam.AddCommand(cmdCamPlay)
	cmdHub.AddCommand(cmdHubPlay)
	cmd.AddCommand(cmdHub, cmdCam)

	if err := cmd.Execute(); err != nil {
		utils.Logger.Fatal().Err(err).Msg("Aborting")
	} else {
		utils.Logger.Info().Msg("Exiting")
	}
}
