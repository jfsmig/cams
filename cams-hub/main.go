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

func run(ctx context.Context, cancel context.CancelFunc, cfg HubConfig) error {
	defer cancel()

	wg := sync.WaitGroup{}
	reg := NewHub(ctx, cancel, &wg)

	wg.Add(1)
	go reg.Run(cfg.Listen)
	wg.Wait()

	return nil
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

			cfg := HubConfig{
				Listen: defaultListenAddr,
				Tls: TLSConfig{
					defaultTLSPathCRT,
					defaultTLSPathKey,
				},
			}

			return run(ctx, cancel, cfg)
		},
	}

	if err := cmd.Execute(); err != nil {
		utils.Logger.Fatal().Err(err).Msg("Aborting")
	} else {
		utils.Logger.Info().Msg("Exiting")
	}
}
