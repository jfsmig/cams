// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	pb2 "github.com/jfsmig/cams/go/api/pb"
	utils2 "github.com/jfsmig/cams/go/utils"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"net"
)

func main() {
	cmd := &cobra.Command{
		Use:   "hub",
		Short: "Cams Hub",
		Long:  "Hub / Upstream for Cams Agent",
		//Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			cfg := utils2.ServerConfig{
				ListenAddr: "127.0.0.1:6000",
				PathCrt:    "",
				PathKey:    "",
			}
			// FIXME(jfs): load an external configuration file or CLI options
			return runHub(ctx, cfg)
		},
	}

	if err := cmd.Execute(); err != nil {
		utils2.Logger.Fatal().Err(err).Msg("Aborting")
	} else {
		utils2.Logger.Info().Msg("Exiting")
	}
}

func runHub(ctx context.Context, config utils2.ServerConfig) error {
	hub := &grpcHub{
		config: config,
	}

	utils2.Logger.Info().Str("action", "start").Msg("hub")

	var cnx *grpc.Server
	var err error

	if len(config.PathCrt) <= 0 || len(config.PathKey) <= 0 {
		cnx, err = hub.config.ServeInsecure()
	} else {
		cnx, err = hub.config.ServeTLS()
	}
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", hub.config.ListenAddr)
	if err != nil {
		return err
	}

	pb2.RegisterRegistrarServer(cnx, hub)
	pb2.RegisterControllerServer(cnx, hub)
	pb2.RegisterViewerServer(cnx, hub)

	hub.registrar = NewRegistrarInMem()

	return cnx.Serve(listener)
}
