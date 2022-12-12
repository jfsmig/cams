// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"github.com/jfsmig/cams/go/utils"
	"github.com/juju/errors"
	"github.com/spf13/cobra"
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
			return play(args[0], args[1])
		},
	}

	cmd.AddCommand(play)

	if err := cmd.Execute(); err != nil {
		utils.Logger.Fatal().Err(err).Msg("Aborting")
	} else {
		utils.Logger.Info().Msg("Exiting")
	}
}
