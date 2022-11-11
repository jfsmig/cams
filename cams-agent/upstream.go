// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"github.com/jfsmig/cams/api/pb"
	"github.com/jfsmig/cams/utils"
	"github.com/jfsmig/go-bags"
	"go.nanomsg.org/mangos/v3/protocol"
	"go.nanomsg.org/mangos/v3/protocol/pull"
	"google.golang.org/grpc"
	"sync"
	"time"

	"github.com/aler9/gortsplib"
)

type upstreamAgent struct {
	cfg AgentConfig
	cnx *grpc.ClientConn
	lan *lanAgent

	commands chan string

	observers bags.SortedObj[string, CommandObserver]

	singletonLock sync.Mutex
}

type upstreamMedia struct {
	us      *upstreamAgent
	camID   string
	control chan string
}

type CommandObserver interface {
	PK() string
	Update(camId, cmd string)
}

func NewUpstreamAgent(cfg AgentConfig) *upstreamAgent {
	return &upstreamAgent{
		cfg:           cfg,
		cnx:           nil,
		lan:           nil,
		singletonLock: sync.Mutex{},
		observers:     make([]CommandObserver, 0),
		commands:      make(chan string),
	}
}

const (
	StreamCommandPlay = "p"
	StreamCommandStop = "s"
	StreamCommandUp   = "u"
	StreamCommandDown = "d"
)

func (us *upstreamAgent) AttachCommandObserver(observer CommandObserver) {
	us.observers.Add(observer)
}

func (us *upstreamAgent) DetachCommandObserver(observer CommandObserver) {
	us.observers.Remove(observer.PK())
}

// NotifyCameraExpectation reports a stream expectation to external observers.
// The call is dedicated to inform the camera that their stream is expected
// to be played ASAP
func (us *upstreamAgent) NotifyCameraExpectation(camId, cmd string) {
	for _, observer := range us.observers {
		observer.Update(camId, cmd)
	}
}

func (us *upstreamAgent) Run(ctx context.Context, lan *lanAgent) {
	utils.Logger.Debug().Str("action", "start").Msg("upstream")

	if !us.singletonLock.TryLock() {
		panic("BUG the upstream agent is already running")
	}
	defer us.singletonLock.Unlock()

	for ctx.Err() == nil {
		// FIXME(jfs): implement some increasing exponential back-off
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
	utils.GroupRun(ctx,
		func(c context.Context) { us.runRegistration(c) },
		func(c context.Context) { us.runControl(c) },
		func(c context.Context) { us.runMain(c) })
}

func (us *upstreamAgent) runMain(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case cmd := <-us.commands:
			prefix := cmd[:1]
			cmd = cmd[1:]
			switch prefix {
			case StreamCommandPlay:
				// TODO(jfs): Ensure there is a running cam agent
				// TODO(jfs): Trigger the stream at the agent level
				us.NotifyCameraExpectation(cmd, StreamCommandPlay)
			case StreamCommandStop:
				// TODO(jfs): Ensure there is a running cam agent
				// TODO(jfs): Notify the camera
				us.NotifyCameraExpectation(cmd, StreamCommandStop)
			case StreamCommandUp:
				// TODO(jfs): Ensure there is a running cam agent
			case StreamCommandDown:
				// TODO(jfs): Tear the cam agent down

			}
		}
	}
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
			_, err := client.Register(ctx, &inReq)
			if err != nil {
				utils.Logger.Warn().Err(err).
					Str("action", "register").
					Msg("upstream registration")
				return
			} else {
				utils.Logger.Debug().
					Str("action", "register").
					Msg("upstream registration")
			}
		}
	}
}

func (us *upstreamAgent) runControl(ctx context.Context) {
	utils.Logger.Trace().Str("action", "start").Msg("upstream control")

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
			continue
		}
		srv := gortsplib.Server{}
		srv.Handler = gortsplib.ServerHandlerOnSessionOpenCtx{}

		switch request.Command {
		case pb.StreamCommandType_StreamCommandType_PLAY:
			us.NotifyCameraExpectation(request.StreamID, StreamCommandPlay)
		case pb.StreamCommandType_StreamCommandType_STOP:
			us.NotifyCameraExpectation(request.StreamID, StreamCommandStop)
		}
	}
}

func (us *upstreamAgent) runMedia(ctx context.Context, camID string) {
	um := upstreamMedia{
		us:      us,
		camID:   camID,
		control: make(chan string, 0),
	}
	us.swarm.Run(func(c context.Context) { um.Run(c) })
}

func (us *upstreamAgent) Update(camId string, state CameraState) {
	switch state {
	case CameraOnline:
		us.commands <- StreamCommandUp + camId
	case CameraOffline:
		us.commands <- StreamCommandDown + camId
	}
}

func (us *upstreamAgent) PK() string { return "ua" }

func (um *upstreamMedia) Run(ctx context.Context) {
	utils.Logger.Trace().Str("action", "start").Str("camera", um.camID).Msg("upstream media")

	// Connect to the internal media bridge fed by the camera
	socketRtp, err := pull.NewSocket()
	if err != nil {
		utils.Logger.Warn().Str("action", "north socket").Err(err).Msg("upstream media")
		return
	}
	if err = socketRtp.Dial(makeNorthRtp(um.camID)); err != nil {
		utils.Logger.Warn().Str("action", "north dial").Err(err).Msg("upstream media")
		return
	}

	// Connect to the internal media bridge fed by the camera
	socketRtcp, err := pull.NewSocket()
	if err != nil {
		utils.Logger.Warn().Str("action", "north socket").Err(err).Msg("upstream media")
		return
	}
	if err = socketRtcp.Dial(makeNorthRtcp(um.camID)); err != nil {
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

	utils.GroupRun(ctx,
		func(c context.Context) { upstreamPipeline(c, socketRtp, ctrl, pb.MediaFrameType_FrameType_RTP) },
		func(c context.Context) { upstreamPipeline(c, socketRtcp, ctrl, pb.MediaFrameType_FrameType_RTCP) },
	)
}

func upstreamPipeline(ctx context.Context, sock protocol.Socket, ctrl pb.Controller_MediaUploadClient, frameType pb.MediaFrameType) {
	proto := "rtp"
	if frameType == pb.MediaFrameType_FrameType_RTCP {
		proto = "rtcp"
	}

	for {
		if err := ctx.Err(); err != nil {
			utils.Logger.Warn().Str("action", "interrupted").Err(err).Str("proto", proto).Msg("upstream media")
			return
		}
		msg, err := sock.Recv()
		if err != nil {
			utils.Logger.Warn().Str("action", "consume").Err(err).Msg("upstream media")
			return
		}

		frame := &pb.MediaFrame{
			Type:    frameType,
			Payload: msg,
		}
		if err := ctrl.Send(frame); err != nil {
			utils.Logger.Warn().Str("action", "send").Err(err).Msg("upstream media")
			return
		}
	}
}
