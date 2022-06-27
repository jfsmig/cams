// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"github.com/jfsmig/cams/utils"
	"github.com/spf13/cobra"
	"sync"
)

const (
	defaultListenAddr = "127.0.0.1:6000"
	defaultTLSPathCRT = ""
	defaultTLSPathKey = ""
)

type runnable func(ctx context.Context) error

func runChild(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, cb runnable) error {
	defer cancel()
	defer wg.Done()
	return cb(ctx)
}

func main() {
	cmd := &cobra.Command{
		Use:   "hub",
		Short: "Cams Hub",
		Long:  "Hub / Upstream for OnVif cameras Agent",
		//Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			cfg := utils.ServerConfig{
				ListenAddr: defaultListenAddr,
				PathCrt:    defaultTLSPathCRT,
				PathKey:    defaultTLSPathKey,
			}

			return runHub(ctx, cfg)
		},
	}

	if err := cmd.Execute(); err != nil {
		utils.Logger.Fatal().Err(err).Msg("Aborting")
	} else {
		utils.Logger.Info().Msg("Exiting")
	}
}
