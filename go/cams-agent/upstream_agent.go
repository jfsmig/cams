// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"sync"
	"time"

	"github.com/aler9/gortsplib"
	"github.com/jfsmig/go-bags"
	"github.com/juju/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/jfsmig/cams/go/api/pb"
	"github.com/jfsmig/cams/go/utils"
)

type UpstreamCommandType uint32

type UpstreamCommand struct {
	cmdType  UpstreamCommandType
	streamID string
}

const (
	upstreamAgent_CommandPlay UpstreamCommandType = iota
	upstreamAgent_CommandStop
	upstreamAgent_CamUp
	upstreamAgent_CamDown
	upstreamAgent_CamVanished
)

type StreamExpectation string

const (
	UpstreamAgent_ExpectPlay  StreamExpectation = "play"
	UpstreamAgent_ExpectPause                   = "pause"
)

var (
	ErrNoSuchCamera = errors.New("no such camera")
)

type upstreamAgent struct {
	cfg AgentConfig
	lan *lanAgent

	control       chan UpstreamCommand
	singletonLock sync.Mutex

	observers bags.SortedObj[string, CommandObserver]
	medias    bags.SortedObj[string, UpstreamMedia]

	// One "true" entry means that the hub expects that stream to be played
	expectations map[string]bool
}

type CommandObserver interface {
	PK() string
	UpdateStreamExpectation(camID string, cmd StreamExpectation)
}

func NewUpstreamAgent(cfg AgentConfig) *upstreamAgent {
	return &upstreamAgent{
		cfg:           cfg,
		lan:           nil,
		control:       make(chan UpstreamCommand),
		singletonLock: sync.Mutex{},
		observers:     make([]CommandObserver, 0),
		medias:        make([]UpstreamMedia, 0),
		expectations:  make(map[string]bool),
	}
}

func (us *upstreamAgent) PK() string { return "ua" }

func (us *upstreamAgent) Run(ctx context.Context, lan *lanAgent) {
	utils.Logger.Debug().Str("action", "start").Msg("up")

	if !us.singletonLock.TryLock() {
		panic("BUG the upstream agent is already running")
	}
	defer us.singletonLock.Unlock()

	for ctx.Err() == nil {
		<-time.After(time.Second) // Pause to avoid crazy looping of connection attempts
		us.reconnectAndRerun(ctx, lan)
	}
}

func (us *upstreamAgent) AttachCommandObserver(observer CommandObserver) {
	us.observers.Add(observer)
}

func (us *upstreamAgent) DetachCommandObserver(observer CommandObserver) {
	us.observers.Remove(observer.PK())
}

func (us *upstreamAgent) UpdateCameraState(camID string, state CameraState) {
	switch state {
	case CameraState_Online:
		us.control <- UpstreamCommand{upstreamAgent_CamUp, camID}
	case CameraState_Offline:
		us.control <- UpstreamCommand{upstreamAgent_CamDown, camID}
	}
}

// NotifyCameraExpectation reports a stream expectation to external observers.
// The call is dedicated to inform the camera that their stream is expected
// to be played ASAP
func (us *upstreamAgent) NotifyCameraExpectation(camID string, cmd StreamExpectation) {
	for _, observer := range us.observers {
		observer.UpdateStreamExpectation(camID, cmd)
	}
}

func (us *upstreamAgent) getRegisterPeriod() time.Duration {
	if us.cfg.RegisterPeriod > 0 {
		return time.Duration(us.cfg.RegisterPeriod) * time.Second
	}
	return 30 * time.Second
}

func (us *upstreamAgent) runMain(ctx context.Context, cnx *grpc.ClientConn) {
	utils.Logger.Trace().Str("action", "start").Msg("up")

	registrationNext := time.After(0)
	client := pb.NewRegistrarClient(cnx)

	camSwarm := utils.NewSwarm(ctx)
	defer camSwarm.Wait()
	defer camSwarm.Cancel()

	for {
		select {
		case <-ctx.Done():
			return

		case <-registrationNext:
			registrationNext = time.After(us.getRegisterPeriod())
			ctx2 := metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{
				utils.KeyUser: us.cfg.User,
			}))
			for _, cam := range us.lan.Cameras() {
				inReq := pb.RegisterRequest{
					Id: &pb.StreamId{
						User:   us.cfg.User,
						Stream: cam.ID,
					},
				}
				if _, err := client.Register(ctx2, &inReq); err != nil {
					utils.Logger.Warn().Err(err).
						Str("action", "register").
						Msg("up reg")
					return
				}
			}
		case cmd := <-us.control:
			us.onCommand(cmd, cnx, camSwarm)
		}
	}
}

func (us *upstreamAgent) onCommand(cmd UpstreamCommand, cnx *grpc.ClientConn, camSwarm utils.Swarm) {
	camID := cmd.streamID
	switch cmd.cmdType {
	case upstreamAgent_CommandPlay: // From the hub
		us.expectations[camID] = true
		us.NotifyCameraExpectation(camID, UpstreamAgent_ExpectPlay)
		cam, ok := us.medias.Get(camID)
		if !ok {
			utils.Logger.Warn().Str("cam", camID).Err(ErrNoSuchCamera).Msg("up ctrl")
			// FIXME(jfs): command for an inexistant camera. Maybe need to manage a rogue cloud service
		} else {
			cam.CommandPlay()
		}
	case upstreamAgent_CommandStop: // From the hub
		us.expectations[camID] = false
		us.NotifyCameraExpectation(camID, UpstreamAgent_ExpectPause)
		if cam, ok := us.medias.Get(camID); !ok {
			utils.Logger.Warn().Str("cam", camID).Err(ErrNoSuchCamera).Msg("up ctrl")
			// FIXME(jfs): command for an inexistant camera. Maybe need to manage a rogue cloud service
		} else {
			cam.CommandPause()
		}
	case upstreamAgent_CamUp: // From the lan
		utils.Logger.Info().Str("cmd", "up").Str("cam", camID).Msg("up ctrl")
		if _, ok := us.medias.Get(camID); !ok {
			um := NewUpstreamMedia(camID, us.cfg)
			us.medias.Add(um)
		}
		if us.expectations[camID] {
			um, _ := us.medias.Get(camID)
			camSwarm.Run(func(c context.Context) { um.Run(c, cnx) })
		}
	case upstreamAgent_CamDown: // From the lan
		utils.Logger.Info().Str("cmd", "down").Str("cam", camID).Msg("up ctrl")
		if cam, ok := us.medias.Get(camID); !ok {
			utils.Logger.Warn().Str("cam", camID).Err(ErrNoSuchCamera).Msg("up ctrl")
			// FIXME(jfs): command for an inexistant camera. Need to manage a rogue cloud service
		} else {
			cam.CommandPause()
			us.NotifyCameraExpectation(camID, UpstreamAgent_ExpectPause)
		}
	case upstreamAgent_CamVanished: // From the lan
		utils.Logger.Info().Str("cmd", "vanished").Str("cam", camID).Msg("up ctrl")
		if cam, ok := us.medias.Get(camID); !ok {
			utils.Logger.Warn().Str("cam", camID).Err(ErrNoSuchCamera).Msg("up ctrl")
		} else {
			us.medias.Remove(camID)
			cam.CommandShut()
		}
	default:
		panic("BUG: unexpected command")
	}
}

// runControl polls the control stream and forward them in the command channel
// is the upstreamAgent
func (us *upstreamAgent) runControl(ctx context.Context, cnx *grpc.ClientConn) {
	utils.Logger.Trace().Str("action", "start").Msg("up ctrl")

	client := pb.NewDownstreamClient(cnx)

	ctx = metadata.AppendToOutgoingContext(ctx,
		utils.KeyUser, us.cfg.User)

	ctrl, err := client.Control(ctx)
	if err != nil {
		utils.Logger.Warn().Str("action", "open").Err(err).Msg("up ctrl")
		return
	}

	defer func() {
		if err := ctrl.CloseSend(); err != nil {
			utils.Logger.Warn().Str("action", "close").Err(err).Msg("up ctrl")
		}
	}()

	for {
		request, err := ctrl.Recv()
		if err != nil {
			utils.Logger.Warn().Str("action", "read").Err(err).Msg("up ctrl")
			return
		}
		srv := gortsplib.Server{}
		srv.Handler = gortsplib.ServerHandlerOnSessionOpenCtx{}

		switch request.Command {
		case pb.DownstreamCommandType_DOWNSTREAM_COMMAND_TYPE_PLAY:
			us.control <- UpstreamCommand{upstreamAgent_CommandPlay, request.StreamID}
		case pb.DownstreamCommandType_DOWNSTREAM_COMMAND_TYPE_STOP:
			us.control <- UpstreamCommand{upstreamAgent_CommandStop, request.StreamID}
		}
	}
}

func (us *upstreamAgent) reconnectAndRerun(ctx context.Context, lan *lanAgent) {
	utils.Logger.Trace().Str("action", "restart").Str("endpoint", us.cfg.Upstream.Address).Msg("up")

	cnx, err := utils.DialInsecure(ctx, us.cfg.Upstream.Address)
	if err != nil {
		utils.Logger.Error().Err(err).Str("action", "dial").Msg("up")
		return
	}
	defer cnx.Close()

	us.lan = lan
	utils.GroupRun(ctx,
		func(c context.Context) { us.runControl(c, cnx) },
		func(c context.Context) { us.runMain(c, cnx) })
}
