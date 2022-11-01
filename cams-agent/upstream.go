// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"github.com/jfsmig/cams/api/pb"
	"github.com/jfsmig/cams/utils"
	"go.nanomsg.org/mangos/v3/protocol/pull"
	"google.golang.org/grpc"
	"time"

	"github.com/aler9/gortsplib"
)

type upstreamAgent struct {
	cfg AgentConfig
	cnx *grpc.ClientConn
	lan *lanAgent

	swarm utils.Swarm
}

type upstreamMedia struct {
	us      *upstreamAgent
	camID   string
	control chan string
}

func NewUpstreamAgent(cfg AgentConfig) *upstreamAgent {
	return &upstreamAgent{
		cfg:   cfg,
		cnx:   nil,
		lan:   nil,
		swarm: nil,
	}
}

func (us *upstreamAgent) Run(ctx context.Context, lan *lanAgent) {
	utils.Logger.Debug().Str("action", "start").Msg("upstream")

	for ctx.Err() == nil {
		<-time.After(time.Second) // Pause to avoid crazy looping of connection attempts
		us.reconnectAndRerun(ctx, lan)
	}
}

func (us *upstreamAgent) reconnectAndRerun(ctx context.Context, lan *lanAgent) {
	utils.Logger.Trace().Str("action", "restart").Str("endpoint", us.cfg.Upstream.Address).Msg("upstream")

	cnx, err := utils.DialInsecure(ctx, us.cfg.Upstream.Address)
	if err != nil {
		utils.Logger.Error().Err(err).Str("action", "dial").Msg("upstream")
		return
	}
	defer cnx.Close()

	us.cnx = cnx
	us.lan = lan
	us.swarm = utils.NewSwarm(ctx)

	defer us.swarm.Cancel()
	defer us.swarm.Wait()
	us.swarm.Run(func(c context.Context) { us.runRegistration(c) })
	us.swarm.Run(func(c context.Context) { us.runControl(c) })
}

// runRegistration periodically registers the streams found on the cams that
// have been discovered on the LAN.
func (us *upstreamAgent) runRegistration(ctx context.Context) {
	utils.Logger.Trace().Str("action", "start").Msg("upstream registration")

	ticker := time.Tick(1 * time.Second)

	client := pb.NewRegistrarClient(us.cnx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker:
			inReq := pb.RegisterRequest{
				Id: &pb.StreamId{
					User: us.cfg.User,
					// FIXME(jfs): configure the camera
				},
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
	utils.Logger.Trace().Str("action", "start	").Msg("upstream control")

	client := pb.NewControllerClient(us.cnx)

	ctx = context.WithValue(ctx, "user", us.cfg.User)

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

func (us *upstreamAgent) runMedia(ctx context.Context, camID string) {
	um := &upstreamMedia{
		us:      us,
		camID:   camID,
		control: make(chan string, 0),
	}
	us.swarm.Run(func(c context.Context) { um.Run(c) })
}

func (us *upstreamAgent) UpdateCamera(camId string, state CameraState) {

}

func (us *upstreamAgent) PK() string { return "ua" }

func (um *upstreamMedia) Run(ctx context.Context) {
	utils.Logger.Trace().Str("action", "start").Str("camera", um.camID).Msg("upstream media")

	// Connect to the internal media bridge fed by the camera
	s, err := pull.NewSocket()
	if err != nil {
		utils.Logger.Warn().Str("action", "north socket").Err(err).Msg("upstream media")
		return
	}
	if err = s.Dial(urlNorth + "/" + um.camID); err != nil {
		utils.Logger.Warn().Str("action", "north dial").Err(err).Msg("upstream media")
		return
	}

	// Open a gRPC connection for the upstream
	client := pb.NewControllerClient(um.us.cnx)
	ctx = context.WithValue(ctx, "user", um.us.cfg.User)
	ctx = context.WithValue(ctx, "stream", um.camID)
	ctrl, err := client.MediaUpload(ctx)
	if err != nil {
		utils.Logger.Warn().Str("action", "open").Err(err).Msg("upstream media")
		return
	}

	// Loop on the media frames from the bridge and sent gRPC messages in the upstream
	for {
		msg, err := s.Recv()
		if err != nil {
			utils.Logger.Warn().Str("action", "consume").Err(err).Msg("upstream media")
			return
		}

		frame := utils.MediaDecode(msg)
		if err := ctrl.Send(frame); err != nil {
			utils.Logger.Warn().Str("action", "send").Err(err).Msg("upstream media")
			return
		}
	}
}
