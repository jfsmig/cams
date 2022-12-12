// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"net"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"github.com/jfsmig/cams/go/api/pb"
	"github.com/jfsmig/cams/go/utils"
)

func main() {
	cmd := &cobra.Command{
		Use:   "hub",
		Short: "Cams Hub",
		Long:  "Hub / Upstream for Cams Agent",
		//Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// FIXME(jfs): load an external configuration file or CLI options
			cfg := utils.ServerConfig{
				ListenAddr: "127.0.0.1:6000",
				PathCrt:    "",
				PathKey:    "",
			}

			ctx, cancel := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)
			defer cancel()

			return runHub(ctx, cfg)
		},
	}

	if err := cmd.Execute(); err != nil {
		utils.Logger.Fatal().Err(err).Msg("Aborting")
	} else {
		utils.Logger.Info().Msg("Exiting")
	}
}

func runHub(ctx context.Context, config utils.ServerConfig) error {
	hub := &grpcHub{
		config: config,
	}

	utils.Logger.Info().Str("action", "start").Msg("hub")

	var server *grpc.Server
	var err error

	if len(config.PathCrt) <= 0 || len(config.PathKey) <= 0 {
		server, err = hub.config.ServeInsecure()
	} else {
		server, err = hub.config.ServeTLS()
	}
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", hub.config.ListenAddr)
	if err != nil {
		return err
	}
	defer listener.Close()

	utils.SwarmRun(ctx,
		func(c context.Context) {
			<-c.Done()
			utils.Logger.Info().Str("action", "kill").Msg("hub")
			server.GracefulStop()
		},
		func(c context.Context) {
			pb.RegisterRegistrarServer(server, hub)
			pb.RegisterDownstreamServer(server, hub)
			pb.RegisterViewerServer(server, hub)
			hub.registrar = NewRegistrarInMem()
			if err := server.Serve(listener); err != nil {
				utils.Logger.Warn().Err(err).Msg("grpc")
			}
		},
	)

	return nil
}
