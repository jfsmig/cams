package main

import (
	"context"
	"go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/protocol/pull"
	"go.nanomsg.org/mangos/v3/protocol/push"

	"github.com/jfsmig/cams/go/utils"
)

func makeSouthRtp(camID string) string  { return "inproc://" + camID + "/MS" }
func makeNorthRtp(camID string) string  { return "inproc://" + camID + "/MN" }
func makeSouthRtcp(camID string) string { return "inproc://" + camID + "/CS" }
func makeNorthRtcp(camID string) string { return "inproc://" + camID + "/CN" }

// pipeline bridges two listening socket, from South to North:
// * "south" is a PULL socket (pipeline protocol) where all the LanCamera connect and produce their frames
// * "north" is a PUSH socket (pipeline protocol) destined to produce all the frames to the upstreamAgent
func pipeline(ctx context.Context, urlSouth, urlNorth string) {
	var south, north mangos.Socket
	var err error

	utils.Logger.Info().Str("action", "start").Msg("bridge")

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
