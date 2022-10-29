// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"github.com/jfsmig/cams/utils"
	"github.com/spf13/cobra"
	"go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/protocol/pull"
	"go.nanomsg.org/mangos/v3/protocol/push"
	_ "go.nanomsg.org/mangos/v3/transport/inproc"
	"reflect"
	"runtime"

	"sync"
)

var (
	user         = "admin"
	password     = "ollyhgqo"
	upstreamAddr = "127.0.0.1:6000"
)

const (
	urlSouth = "inproc://s"
	urlNorth = "inproc://n"
)

// swarmRun runs it callback as a part of a swarm, so that when the callback
// returns, the cancel function of called to terminate all the other members
// of the swarm
func swarmRun(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, cb func(context.Context)) {
	defer cancel()
	defer wg.Done()
	utils.Logger.Trace().Str("action", "spawn").Str("cb", runtime.FuncForPC(reflect.ValueOf(cb).Pointer()).Name()).Msg("swarm")
	cb(ctx)
	utils.Logger.Trace().Str("action", "done").Str("cb", runtime.FuncForPC(reflect.ValueOf(cb).Pointer()).Name()).Msg("swarm")
}

func swarm(ctx0 context.Context, callbacks ...func(context.Context)) {
	wg := sync.WaitGroup{}
	ctx, cancel := context.WithCancel(ctx0)

	wg.Add(len(callbacks))
	for _, cb := range callbacks {
		go swarmRun(ctx, cancel, &wg, cb)
	}

	utils.Logger.Trace().Str("action", "wait").Int("count", len(callbacks)).Msg("swarm")
	wg.Wait()
	utils.Logger.Trace().Str("action", "exit").Int("count", len(callbacks)).Msg("swarm")
	cancel()
}

func run(ctx context.Context, cfg AgentConfig) error {
	lan := NewLanAgent()
	lan.Configure(cfg)

	swarm(ctx,
		func(c context.Context) { mediaBridge(c, urlSouth, urlNorth) },
		func(c context.Context) { RunUpstreamAgent(c, upstreamAddr) },
		func(c context.Context) { lan.Run(c) },
	)
	return nil
}

// mediaBridge bridges two listening socket:
// * "south" is a PULL socket (pipeline protocol) where all the LanCamera connect and produce their frames
// * "north" is a PUSH socket (pipeline protocol) destined to produce all the frames to the upstreamAgent
func mediaBridge(ctx context.Context, urlSouth, urlNorth string) {
	var south, north mangos.Socket
	var err error

	utils.Logger.Info().Str("action", "run").Msg("bridge")

	// Establish the two sides of the bridge
	if south, err = pull.NewSocket(); err != nil {
		utils.Logger.Error().Str("action", "south socket").Err(err).Msg("bridge")
		return
	}
	defer south.Close()
	if err = south.Listen(urlSouth); err != nil {
		utils.Logger.Error().Str("action", "south listen").Err(err).Msg("bridge")
		return
	}

	if north, err = push.NewSocket(); err != nil {
		utils.Logger.Error().Str("action", "north socket").Err(err).Msg("bridge")
		return
	}
	defer north.Close()
	if err = south.Listen(urlNorth); err != nil {
		utils.Logger.Error().Str("action", "north listen").Err(err).Msg("bridge")
		return
	}

	// Run the bridging code
	for {
		if ctx.Err() != nil {
			utils.Logger.Info().Str("action", "shutdown").Err(err).Msg("bridge")
			return
		}

		var msg []byte
		if msg, err = south.Recv(); err != nil {
			utils.Logger.Error().Str("action", "south consume").Err(err).Msg("bridge")
			return
		}
		if err = north.Send(msg); err != nil {
			utils.Logger.Error().Str("action", "north produce").Err(err).Msg("bridge")
			return
		}
	}
}

func main() {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Cams Agent",
		Long:  "LAN agent for OnVif cameras",
		//Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := AgentConfig{
				DiscoverPatterns: []string{"!lo", "!docker.*", ".*"},
			}
			return run(context.Background(), cfg)
		},
	}

	if err := cmd.Execute(); err != nil {
		utils.Logger.Fatal().Err(err).Msg("Aborting")
	} else {
		utils.Logger.Info().Msg("Exiting")
	}
}
