// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"bytes"
	"context"
	"github.com/jfsmig/cams/utils"
	"go.nanomsg.org/mangos/v3/protocol/pull"
	"google.golang.org/grpc"
	"time"

	"github.com/aler9/gortsplib"
	"github.com/jfsmig/cams/proto"
)

func RunUpstreamAgent(ctx context.Context, addr string) {
	utils.Logger.Info().Str("action", "run").Msg("upstream")

	for ctx.Err() == nil {
		<-time.After(time.Second) // Pause to avoid crazy looping of connection attempts
		reconnectAndRerun(ctx, addr)
	}
}

type upstreamAgent struct {
	cnx *grpc.ClientConn
}

func reconnectAndRerun(ctx context.Context, addr string) {
	utils.Logger.Info().Str("action", "restart").Str("endpoint", addr).Msg("upstream")

	cnx, err := utils.DialGrpc(ctx, addr)
	if err != nil {
		utils.Logger.Error().Err(err).Str("action", "dial").Msg("upstream")
		return
	}
	defer cnx.Close()

	cus := upstreamAgent{
		cnx: cnx,
	}

	swarm(ctx,
		func(c context.Context) { cus.runRegistration(c) },
		func(c context.Context) { cus.runControl(c) },
		func(c context.Context) { cus.runStream(c) })
}

// runRegistration periodically registers the streams found on the cams that
// have been discovered on the LAN.
func (us *upstreamAgent) runRegistration(ctx context.Context) {
	utils.Logger.Info().Str("action", "run").Msg("upstream registration")

	ticker := time.Tick(1 * time.Second)

	client := proto.NewRegistrarClient(us.cnx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker:
			inReq := proto.RegisterRequest{
				Id: &proto.StreamId{},
			}
			inRep, err := client.Register(ctx, &inReq)
			if err != nil {
				utils.Logger.Warn().Err(err).
					Str("action", "register").
					Msg("upstream registration")
				return
			} else {
				utils.Logger.Debug().
					Uint32("status", inRep.Status.Code).
					Str("msg", inRep.Status.Status).
					Str("action", "register").
					Msg("upstream registration")
			}
		}
	}
}

func (us *upstreamAgent) runControl(ctx context.Context) {
	utils.Logger.Info().Str("action", "run").Msg("upstream control")

	client := proto.NewControllerClient(us.cnx)
	ctrl, err := client.Control(ctx)
	if err != nil {
		utils.Logger.Warn().Str("action", "open").Err(err).Msg("upstream control")
		return
	}

	defer func() {
		if err := ctrl.CloseSend(); err != nil {
			utils.Logger.Warn().Str("action", "close").Err(err).Msg("upstream control")
		}
	}()

	for {
		request, err := ctrl.Recv()
		if err != nil {
			utils.Logger.Warn().Str("action", "read").Err(err).Msg("upstream control")
			return
		}
		srv := gortsplib.Server{}
		srv.Handler = gortsplib.ServerHandlerOnSessionOpenCtx{}

		switch {
		case request.GetTeardown() != nil:
			// FIXME(jfs): NYI
		case request.GetPause() != nil:
			// FIXME(jfs): NYI
		case request.GetPlay() != nil:
			// FIXME(jfs): NYI
		}
	}
}

func (us *upstreamAgent) runStream(ctx context.Context) {
	client := proto.NewControllerClient(us.cnx)
	ctrl, err := client.MediaUpload(ctx)
	if err != nil {
		utils.Logger.Warn().Str("action", "open").Err(err).Msg("upstream media")
		return
	}

	s, err := pull.NewSocket()
	if err != nil {
		utils.Logger.Warn().Str("action", "north socket").Err(err).Msg("upstream media")
		return
	}
	if err = s.Dial(urlNorth); err != nil {
		utils.Logger.Warn().Str("action", "north dial").Err(err).Msg("upstream media")
		return
	}

	for {
		msg, err := s.Recv()
		if err != nil {
			utils.Logger.Warn().Str("action", "consume").Err(err).Msg("upstream media")
			return
		}

		// Extract the identifiers of the owner and the stream itself.
		offsetUser := 0
		offsetID := bytes.IndexByte(msg[offsetUser:], 0)
		if offsetID < 0 {
			panic("invalid internal msg (id)")
		}
		offsetID++
		offsetFrame := bytes.IndexByte(msg[offsetID:], 0)
		if offsetFrame < 0 {
			panic("invalid internal msg (frame)")
		}
		offsetFrame++

		// Copy the message
		frame := &proto.MediaFrame{}
		frame.Id = &proto.StreamId{
			User:   string(msg[offsetUser : offsetID-1]),
			Stream: string(msg[offsetID : offsetFrame-1]),
		}
		frame.Payload = msg

		if err := ctrl.Send(frame); err != nil {
			utils.Logger.Warn().Str("action", "send").Err(err).Msg("upstream media")
			return
		}
	}
}
