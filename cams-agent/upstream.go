// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"github.com/jfsmig/cams/utils"
	"google.golang.org/grpc"
	"sync"
	"time"

	"github.com/aler9/gortsplib"
	"github.com/jfsmig/cams/proto"
)

func RunUpstreamAgent(ctx context.Context, addr string) {
	for ctx.Err() != nil {
		<-time.After(time.Second) // Pause to avoid crazy looping of connection attempts
		reconnectAndRerun(ctx, addr)
	}
}

type connectedUpstreamAgent struct {
	cnx *grpc.ClientConn
}

func reconnectAndRerun(ctx0 context.Context, addr string) {
	ctx, cancel := context.WithCancel(ctx0)
	defer cancel()
	wg := sync.WaitGroup{}

	cnx, err := utils.DialGrpc(ctx, addr)
	if err != nil {
		utils.Logger.Error().Err(err).Str("action", "dial").Msg("upstream")
	}
	defer cnx.Close()

	cus := connectedUpstreamAgent{
		cnx: cnx,
	}

	wg.Add(3)
	go swarmRun(ctx, cancel, &wg, func(c context.Context) { cus.runRegistration(c) })
	go swarmRun(ctx, cancel, &wg, func(c context.Context) { cus.runControl(c) })
	go swarmRun(ctx, cancel, &wg, func(c context.Context) { cus.runStream(c) })
	wg.Wait()
}

// runRegistration periodically registers the streams found on the cams that
// have been discovered on the LAN.
func (us *connectedUpstreamAgent) runRegistration(ctx context.Context) {
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
				utils.Logger.Warn().Str("action", "register").Err(err).Msg("register")
				return
			} else {
				utils.Logger.Debug().
					Uint32("status", inRep.Status.Code).
					Str("msg", inRep.Status.Status).
					Str("action", "do").
					Msg("register")
			}
		}
	}
}

func (us *connectedUpstreamAgent) runControl(ctx context.Context) {
	client := proto.NewControllerClient(us.cnx)
	ctrl, err := client.Control(ctx)
	if err != nil {
		utils.Logger.Warn().Str("action", "open").Err(err).Msg("control")
		return
	}

	defer func() {
		if err := ctrl.CloseSend(); err != nil {
			utils.Logger.Warn().Str("action", "close").Err(err).Msg("control")
		}
	}()

	for {
		request, err := ctrl.Recv()
		if err != nil {
			utils.Logger.Warn().Str("action", "read").Err(err).Msg("control")
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

func (us *connectedUpstreamAgent) runStream(ctx context.Context) {
	client := proto.NewControllerClient(us.cnx)
	ctrl, err := client.MediaUpload(ctx)
	if err != nil {
		utils.Logger.Warn().Str("action", "open").Err(err).Msg("media")
		return
	}
	for {
		client := gortsplib.Server{}

		frame := &proto.MediaFrame{}
		err := ctrl.Send(frame)
		if err != nil {
			utils.Logger.Warn().Str("action", "send").Err(err).Msg("media")
			return
		}
	}
}
