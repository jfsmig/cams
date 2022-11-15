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

type UpstreamCommand string

const (
	upstreamAgent_CommandPlay string = "p"
	upstreamAgent_CommandStop        = "s"
	upstreamAgent_CamUp              = "u"
	upstreamAgent_CamDown            = "d"
	upstreamAgent_CamVanished        = "e"
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

	control       chan string
	singletonLock sync.Mutex

	observers bags.SortedObj[string, CommandObserver]
	medias    bags.SortedObj[string, UpstreamMedia]
}

type CommandObserver interface {
	PK() string
	UpdateStreamExpectation(camID string, cmd StreamExpectation)
}

func NewUpstreamAgent(cfg AgentConfig) *upstreamAgent {
	return &upstreamAgent{
		cfg:           cfg,
		lan:           nil,
		singletonLock: sync.Mutex{},
		observers:     make([]CommandObserver, 0),
		control:       make(chan string),
	}
}

func (us *upstreamAgent) PK() string { return "ua" }

func (us *upstreamAgent) Run(ctx context.Context, lan *lanAgent) {
	utils.Logger.Debug().Str("action", "start").Msg("upstream")

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
		us.control <- upstreamAgent_CamUp + camID
	case CameraState_Offline:
		us.control <- upstreamAgent_CamDown + camID
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

func (us *upstreamAgent) runMain(ctx context.Context, cnx *grpc.ClientConn) {
	utils.Logger.Trace().Str("action", "start").Msg("upstream")

	registrationTicker := time.Tick(5 * time.Second)
	client := pb.NewRegistrarClient(cnx)

	camSwarm := utils.NewSwarm(ctx)
	defer camSwarm.Wait()
	defer camSwarm.Cancel()

	for {
		select {
		case <-ctx.Done():
			return

		case <-registrationTicker:
			inReq := pb.RegisterRequest{
				Id: &pb.StreamId{
					User: us.cfg.User,
					// FIXME(jfs): configure the camera
				},
			}

			ctx = metadata.AppendToOutgoingContext(ctx,
				utils.KeyUser, us.cfg.User)

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

		case cmd := <-us.control:
			us.onCommand(cmd, cnx, camSwarm)
		}
	}
}

func (us *upstreamAgent) onCommand(cmd string, cnx *grpc.ClientConn, camSwarm utils.Swarm) {
	prefix := cmd[:1]
	camID := cmd[1:]

	switch prefix {
	case upstreamAgent_CommandPlay:
		cam, ok := us.medias.Get(camID)
		if !ok {
			utils.Logger.Warn().Str("cam", camID).Err(ErrNoSuchCamera).Msg("upstream control")
			// FIXME(jfs): command for an inexistant camera. Need to manage a rogue cloud service
		} else {
			cam.CommandPlay()
			us.NotifyCameraExpectation(camID, UpstreamAgent_ExpectPlay)
		}
	case upstreamAgent_CommandStop:
		if cam, ok := us.medias.Get(camID); !ok {
			utils.Logger.Warn().Str("cam", camID).Err(ErrNoSuchCamera).Msg("upstream control")
			// FIXME(jfs): command for an inexistant camera. Need to manage a rogue cloud service
		} else {
			cam.CommandPause()
			us.NotifyCameraExpectation(camID, UpstreamAgent_ExpectPause)
		}
	case upstreamAgent_CamUp:
		if _, ok := us.medias.Get(camID); !ok {
			um := NewUpstreamMedia(camID, us.cfg)
			us.medias.Add(um)
			camSwarm.Run(func(c context.Context) { um.Run(c, cnx) })
		}
	case upstreamAgent_CamDown:
		if cam, ok := us.medias.Get(camID); !ok {
			utils.Logger.Warn().Str("cam", camID).Err(ErrNoSuchCamera).Msg("upstream control")
			// FIXME(jfs): command for an inexistant camera. Need to manage a rogue cloud service
		} else {
			cam.CommandPause()
			us.NotifyCameraExpectation(camID, UpstreamAgent_ExpectPause)
		}
	case upstreamAgent_CamVanished:
		if cam, ok := us.medias.Get(camID); !ok {
			utils.Logger.Warn().Str("cam", camID).Err(ErrNoSuchCamera).Msg("upstream control")
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
	utils.Logger.Trace().Str("action", "start").Msg("upstream control")

	client := pb.NewControllerClient(cnx)

	ctx = metadata.AppendToOutgoingContext(ctx,
		utils.KeyUser, us.cfg.User)

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

		switch request.Command {
		case pb.StreamCommandType_StreamCommandType_PLAY:
			us.control <- upstreamAgent_CommandPlay + request.StreamID
		case pb.StreamCommandType_StreamCommandType_STOP:
			us.control <- upstreamAgent_CommandStop + request.StreamID
		}
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

	us.lan = lan
	utils.GroupRun(ctx,
		func(c context.Context) { us.runControl(c, cnx) },
		func(c context.Context) { us.runMain(c, cnx) })
}
