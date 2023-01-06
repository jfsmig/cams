// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package lan

import (
	"context"
	"go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/protocol/pull"
	"go.nanomsg.org/mangos/v3/protocol/push"
	"time"

	"github.com/jfsmig/cams/go/utils"
)

// pipeline bridges two listening socket, from South to North:
// * "south" is a PULL socket (pipeline protocol) where one Camera connects and produce its frames
// * "north" is a PUSH socket (pipeline protocol) destined to produce all the frames to the upstream.Agent for a given Camera
func pipeline(ctx context.Context, urlSouth, urlNorth string) {
	var south, north mangos.Socket
	var err error

	utils.Logger.Info().Str("action", "start").Msg("lan-br")

	// Establish the two sides of the bridge
	if south, err = pull.NewSocket(); err != nil {
		utils.Logger.Error().Str("action", "south socket").Err(err).Msg("lan-br")
		return
	}
	defer south.Close()
	if err = south.Listen(urlSouth); err != nil {
		utils.Logger.Error().Str("action", "south listen").Err(err).Msg("lan-br")
		return
	}

	if north, err = push.NewSocket(); err != nil {
		utils.Logger.Error().Str("action", "north socket").Err(err).Msg("lan-br")
		return
	}
	defer north.Close()
	if err = south.Listen(urlNorth); err != nil {
		utils.Logger.Error().Str("action", "north listen").Err(err).Msg("lan-br")
		return
	}

	north.SetOption(mangos.OptionBestEffort, true)
	north.SetOption(mangos.OptionNoDelay, true)
	north.SetOption(mangos.OptionSendDeadline, true)

	// Run the bridging code
	for {
		if ctx.Err() != nil {
			utils.Logger.Info().Str("action", "shutdown").Err(err).Msg("lan-br")
			return
		}

		var msg []byte
		if msg, err = south.Recv(); err != nil {
			utils.Logger.Error().Str("action", "south consume").Err(err).Msg("lan-br")
			return
		}
		north.SetOption(mangos.OptionSendDeadline, time.Duration(-1))
		if err = north.Send(msg); err != nil {
			utils.Logger.Error().Str("action", "north produce").Err(err).Msg("lan-br")
			return
		}
	}
}
