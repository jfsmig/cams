// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"github.com/aler9/gortsplib"
	pb2 "github.com/jfsmig/cams/go/api/pb"
	"github.com/jfsmig/cams/go/camera"
	utils2 "github.com/jfsmig/cams/go/utils"
	"github.com/juju/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"sync"
	"time"
)

type upstreamCommandType uint32

type upstreamCommand struct {
	cmdType  upstreamCommandType
	streamID string
}

const (
	upstreamAgent_CommandPlay upstreamCommandType = iota
	upstreamAgent_CommandStop
)

var (
	ErrNoSuchCamera = errors.New("no such camera")
)

type upstreamAgent struct {
	cfg AgentConfig
	lan *Agent

	control       chan upstreamCommand
	singletonLock sync.Mutex
}

func NewUpstreamAgent(cfg AgentConfig) *upstreamAgent {
	return &upstreamAgent{
		cfg:           cfg,
		lan:           nil,
		control:       make(chan upstreamCommand),
		singletonLock: sync.Mutex{},
	}
}

// PK also implements a CameraObserver
func (us *upstreamAgent) PK() string { return "ua" }

func (us *upstreamAgent) Run(ctx context.Context, lan *Agent) {
	utils2.Logger.Debug().Str("action", "start").Msg("up")

	if !us.singletonLock.TryLock() {
		panic("BUG the upstream agent is already running")
	}
	defer us.singletonLock.Unlock()

	for ctx.Err() == nil {
		<-time.After(time.Second) // Pause to avoid crazy looping of connection attempts
		us.reconnectAndRerun(ctx, lan)
	}
}

func (us *upstreamAgent) getRegisterPeriod() time.Duration {
	if us.cfg.RegisterPeriod > 0 {
		return time.Duration(us.cfg.RegisterPeriod) * time.Second
	}
	return 30 * time.Second
}

func (us *upstreamAgent) runMain(ctx context.Context, cnx *grpc.ClientConn) error {
	utils2.Logger.Trace().Str("action", "start").Msg("up")

	registrationNext := time.After(0)
	client := pb2.NewRegistrarClient(cnx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-registrationNext:
			registrationNext = time.After(us.getRegisterPeriod())
			ctx2 := metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{
				utils2.KeyUser: us.cfg.User,
			}))
			for _, cam := range us.lan.Cameras() {
				inReq := pb2.RegisterRequest{
					Id: &pb2.StreamId{
						User:   us.cfg.User,
						Stream: cam.ID,
					},
				}
				if _, err := client.Register(ctx2, &inReq); err != nil {
					return errors.Annotate(err, "register")
				}
			}
		case cmd := <-us.control:
			if err := us.onCommand(cmd); err != nil {
				return errors.Annotate(err, "control command")
			}
		}
	}
}

func (us *upstreamAgent) onCommand(cmd upstreamCommand) error {
	camID := cmd.streamID
	switch cmd.cmdType {
	case upstreamAgent_CommandPlay: // From the hub
		return us.lan.UpdateStreamExpectation(camID, camera.CamCommandPlay)
	case upstreamAgent_CommandStop: // From the hub
		return us.lan.UpdateStreamExpectation(camID, camera.CamCommandPause)
	default:
		return errors.New("BUG: unexpected command")
	}
}

// runControl polls the control stream and forward them in the command channel
// is the upstreamAgent
func (us *upstreamAgent) runControl(ctx context.Context, cnx *grpc.ClientConn) error {
	utils2.Logger.Trace().Str("action", "start").Msg("up ctrl")

	client := pb2.NewControllerClient(cnx)

	ctx = metadata.AppendToOutgoingContext(ctx,
		utils2.KeyUser, us.cfg.User)

	ctrl, err := client.Control(ctx)
	if err != nil {
		return errors.Annotate(err, "control open")
	}

	defer func() {
		if err := ctrl.CloseSend(); err != nil {
			utils2.Logger.Warn().Str("action", "close").Err(err).Msg("control close")
		}
	}()

	for {
		request, err := ctrl.Recv()
		if err != nil {
			return errors.Annotate(err, "control recv")
		}
		srv := gortsplib.Server{}
		srv.Handler = gortsplib.ServerHandlerOnSessionOpenCtx{}

		switch request.Command {
		case pb2.DownstreamCommandType_DOWNSTREAM_COMMAND_TYPE_PLAY:
			us.control <- upstreamCommand{upstreamAgent_CommandPlay, request.StreamID}
		case pb2.DownstreamCommandType_DOWNSTREAM_COMMAND_TYPE_STOP:
			us.control <- upstreamCommand{upstreamAgent_CommandStop, request.StreamID}
		}
	}
}

func (us *upstreamAgent) reconnectAndRerun(ctx context.Context, lan *Agent) {
	utils2.Logger.Trace().Str("action", "restart").Str("endpoint", us.cfg.UpstreamControl.Address).Msg("up")

	cnx, err := utils2.DialInsecure(ctx, us.cfg.UpstreamControl.Address)
	if err != nil {
		utils2.Logger.Error().Err(err).Str("action", "dial").Msg("up")
		return
	}
	defer cnx.Close()

	us.lan = lan
	utils2.GroupRun(ctx,
		func(c context.Context) {
			if err := us.runControl(c, cnx); err != nil {
				utils2.Logger.Warn().Err(err).Msg("upstream control error")
			}
		},
		func(c context.Context) {
			if err := us.runMain(c, cnx); err != nil {
				utils2.Logger.Warn().Err(err).Msg("upstream error")
			}
		})
}
