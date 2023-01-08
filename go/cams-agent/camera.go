// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"github.com/jfsmig/cams/go/api/pb"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/metadata"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aler9/gortsplib"
	"github.com/aler9/gortsplib/pkg/url"
	"github.com/jfsmig/onvif/sdk"
	"github.com/juju/errors"

	"github.com/jfsmig/cams/go/utils"
)

type CamAgentState uint32
type CamCommand uint32

const (
	// Agent not running
	CamAgentOff CamAgentState = iota

	// Agent running but waiting for a Play command
	CamAgentIdle

	// Agent running an streaming media
	CamAgentPlaying

	// Media paused and waiting for subroutines to exit before a return to CamAgentOff
	CamAgentPausing

	// Media paused and waiting for subroutines to exit before a return to CamAgentIdle
	CamAgentResuming
)

const (
	CamCommandPing = iota
	CamCommandPlay
	CamCommandPause
)

// We assume only one stream per camera.
type Camera struct {
	agent *Agent

	ID            string
	generation    uint32
	singletonLock sync.Mutex
	State         CamAgentState

	onvifClient sdk.Appliance
	rtspClient  gortsplib.Client

	requests chan CamCommand

	group utils.Swarm
}

func NewCamera(agent *Agent, appliance sdk.Appliance) *Camera {
	transport := gortsplib.TransportUDP
	return &Camera{
		agent:       agent,
		ID:          appliance.GetUUID(),
		generation:  0,
		onvifClient: appliance,
		rtspClient: gortsplib.Client{
			ReadTimeout:           5 * time.Second,
			WriteTimeout:          5 * time.Second,
			RedirectDisable:       true,
			AnyPortEnable:         true,
			Transport:             &transport,
			InitialUDPReadTimeout: 3 * time.Second,
		},
		requests: make(chan CamCommand, 1),
	}
}

func runCam(cam *Camera) utils.SwarmFunc {
	return func(ctx context.Context) { cam.Run(ctx) }
}

func (cam *Camera) Run(ctx context.Context) {
	if !cam.singletonLock.TryLock() {
		panic("BUG singleton only")
	}
	defer cam.singletonLock.Unlock()

	if cam.State != CamAgentOff {
		panic("BUG: unexpected camera agent state")
	}
	cam.State = CamAgentIdle

	defer func() {
		cam.group = nil
		cam.State = CamAgentOff
		close(cam.requests)
		cam.requests = nil
	}()

	transport := gortsplib.TransportUDP
	cam.rtspClient.Transport = &transport

	for {
		select {
		case <-ctx.Done():
			cam.group.Cancel()
			cam.group.Wait()
			return
		case cmd := <-cam.requests:
			switch cmd {
			case CamCommandPlay:
				cam.onCmdPlay(ctx)
			case CamCommandPause:
				cam.onCmdStop()
			case CamCommandPing:
				cam.onCmdPing(ctx)
			}
		}
	}
}

func (cam *Camera) PK() string  { return cam.ID }
func (cam *Camera) Ping()       { cam.requests <- CamCommandPing }
func (cam *Camera) PlayStream() { cam.requests <- CamCommandPlay }
func (cam *Camera) StopStream() { cam.requests <- CamCommandPause }

func (cam *Camera) warn(err error) *zerolog.Event {
	return utils.Logger.Warn().Str("url", cam.ID).Err(err)
}

func (cam *Camera) debug() *zerolog.Event {
	return utils.Logger.Trace().Str("url", cam.ID)
}

func (cam *Camera) runStream(ctx context.Context) {
	for ctx.Err() == nil {
		cam.debug().Msg("cam stream starting")
		err := cam.runStreamOnce(ctx)
		if err != nil {
			cam.debug().Err(err).Msg("cam stream aborted")
		} else {
			// Avoid a crazy loop
			time.Sleep(time.Second)
		}
	}
	cam.debug().Str("url", cam.ID).Msg("cam stream exiting")
}

func (cam *Camera) runStreamOnce(ctx context.Context) error {
	var ctrl pb.Downstream_MediaUploadClient
	var sourceUrl *url.URL
	var err error

	// Prepare the camera side
	sourceUrl, err = cam.queryMediaUrl(ctx)
	if err != nil {
		return errors.Annotate(err, "queryMediaUrl")
	}

	cam.debug().Str("url", sourceUrl.Host).Msg("cam streaming")

	if err = cam.rtspClient.Start(sourceUrl.Scheme, sourceUrl.Host); err != nil {
		return errors.Annotate(err, "start")
	}
	defer func() { _ = cam.rtspClient.Close() }()

	tracks, trackUrl, _, err := cam.rtspClient.Describe(sourceUrl)
	if err != nil {
		return errors.Annotate(err, "describe")
	}

	// Prepare the upstream side
	cnx, err := utils.DialInsecure(ctx, cam.agent.Config.UpstreamMedia.Address)
	if err != nil {
		return errors.Annotate(err, "dial")
	}
	defer cnx.Close()

	client := pb.NewDownstreamClient(cnx)
	ctx = metadata.AppendToOutgoingContext(ctx,
		utils.KeyUser, cam.agent.Config.User,
		utils.KeyStream, cam.PK())
	ctrl, err = client.MediaUpload(ctx)
	if err != nil {
		return errors.Annotate(err, "open")
	}

	// We need a way to break the current goroutine that is just waiting for
	// termination notifications on channels
	localError := make(chan error, 2)
	defer close(localError)
	// We need a way to prevent the rtsp client to notify for errors when the
	// localError channel has been closed.
	var localStop atomic.Bool

	// We need a way to detect the inactivity of the camera.
	// So we check every second that at least one packet has been seen.
	activityJiffies := atomic.Uint32{}
	activityCheckTicker := time.Tick(2 * time.Second)

	cam.rtspClient.OnPacketRTP = func(pkt *gortsplib.ClientOnPacketRTPCtx) {
		activityJiffies.Add(1)
		b, err2 := pkt.Packet.Marshal()
		if err2 == nil {
			frame := &pb.DownstreamMediaFrame{
				Type:    pb.DownstreamMediaFrameType_DOWNSTREAM_MEDIA_FRAME_TYPE_RTP,
				Payload: b,
			}
			err2 = ctrl.Send(frame)
		}
		if err2 != nil {
			utils.Logger.Warn().Str("action", "send").Err(err2).Msg("up media")
			if !localStop.Swap(true) {
				localError <- err2
			}
		}
	}

	cam.rtspClient.OnPacketRTCP = func(pkt *gortsplib.ClientOnPacketRTCPCtx) {
		activityJiffies.Add(1)
		b, err2 := pkt.Packet.Marshal()
		if err2 == nil {
			frame := &pb.DownstreamMediaFrame{
				Type:    pb.DownstreamMediaFrameType_DOWNSTREAM_MEDIA_FRAME_TYPE_RTCP,
				Payload: b,
			}
			err2 = ctrl.Send(frame)
		}
		if err2 != nil {
			utils.Logger.Warn().Str("action", "send").Err(err2).Msg("up media")
			if !localStop.Swap(true) {
				localError <- err2
			}
		}
	}

	// Spawn goroutines that will consume the camera stream
	err = cam.rtspClient.SetupAndPlay(tracks[0:1], trackUrl)
	if err != nil {
		return errors.Annotate(err, "setupAndPlay")
	}

	for activityPrevious := activityJiffies.Load(); ; {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err = <-localError:
			return err
		case <-activityCheckTicker:
			activityCurrent := activityJiffies.Load()
			if activityCurrent == activityPrevious {
				localStop.Store(true)
				return errors.New("timeout")
			}
			activityPrevious = activityCurrent
		}
	}
}

func (cam *Camera) queryMediaUrl(ctx context.Context) (*url.URL, error) {
	streamURI := cam.onvifClient.FetchStreamURI(ctx)
	sourceUrl, err := url.Parse(streamURI)
	if err != nil {
		return nil, errors.Annotate(err, "parse")
	}
	return sourceUrl, nil
}

func (cam *Camera) onCmdPlay(ctx context.Context) {
	switch cam.State {
	case CamAgentOff:
		panic("BUG unexpected state")
	case CamAgentPausing, CamAgentResuming:
		cam.State = CamAgentResuming
		if cam.group.Count() > 0 {
			return
		}
		cam.State = CamAgentIdle
		fallthrough
	case CamAgentIdle:
		cam.State = CamAgentPlaying
		cam.group = utils.NewGroup(ctx)
		cam.group.Run(func(c context.Context) { cam.runStream(c) })
		cam.debug().Msg("camera restarted")
		fallthrough
	case CamAgentPlaying:
		// No-Op
	default:
		panic("BUG invalid state")
	}
}

func (cam *Camera) onCmdStop() {
	switch cam.State {
	case CamAgentOff:
		panic("BUG unexpected state")
	case CamAgentPlaying, CamAgentResuming:
		// Trigger a stop of the coroutines
		cam.group.Cancel()
		cam.State = CamAgentPausing
		fallthrough
	case CamAgentPausing:
		// Wait for the stop to finish
		if cam.group.Count() > 0 {
			return
		}
		cam.group = nil
		cam.State = CamAgentIdle
		fallthrough
	case CamAgentIdle:
		// No-Op
	default:
		panic("BUG invalid state")
	}
}

func (cam *Camera) onCmdPing(ctx context.Context) {
	switch cam.State {
	case CamAgentOff:
		panic("BUG unexpected state")
	case CamAgentIdle, CamAgentPlaying:
		// No-Op
	case CamAgentPausing:
		if cam.group.Count() > 0 {
			return
		}
		cam.group = nil
		cam.State = CamAgentIdle
	case CamAgentResuming:
		if cam.group.Count() > 0 {
			return
		}
		cam.State = CamAgentIdle
		cam.onCmdPlay(ctx)
	default:
		panic("BUG invalid state")
	}
}
