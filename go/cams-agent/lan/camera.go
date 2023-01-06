// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package lan

import (
	"context"
	"github.com/jfsmig/cams/go/cams-agent/common"
	"go.nanomsg.org/mangos/v3"
	"log"
	"sync"
	"time"

	"github.com/aler9/gortsplib"
	"github.com/aler9/gortsplib/pkg/url"
	"github.com/jfsmig/onvif/sdk"
	"github.com/juju/errors"
	"go.nanomsg.org/mangos/v3/protocol/push"
	_ "go.nanomsg.org/mangos/v3/transport/inproc"

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
	CamCommandExit CamCommand = iota
	CamCommandPing
	CamCommandPlay
	CamCommandPause
)

// We assume only one stream per camera.
type Camera struct {
	ID            string
	generation    uint32
	singletonLock sync.Mutex
	State         CamAgentState

	onvifClient sdk.Appliance
	rtspClient  gortsplib.Client

	requests chan CamCommand

	group utils.Swarm
}

func NewCamera(appliance sdk.Appliance) *Camera {
	transport := gortsplib.TransportUDP
	return &Camera{
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
			cam.onCmdExit()
			return
		case cmd := <-cam.requests:
			switch cmd {
			case CamCommandExit:
				cam.onCmdExit()
				return
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
func (cam *Camera) Exit()       { cam.requests <- CamCommandExit }
func (cam *Camera) PlayStream() { cam.requests <- CamCommandPlay }
func (cam *Camera) StopStream() { cam.requests <- CamCommandPause }

func (cam *Camera) runStream(ctx context.Context) {
	for ctx.Err() == nil {
		utils.Logger.Debug().Str("url", cam.ID).Str("action", "start").Msg("cam")
		err := cam.runStreamOnce(ctx)
		if err != nil {
			utils.Logger.Warn().Str("url", cam.ID).Str("action", "done").Err(err).Msg("cam")
		} else {
			// Avoid a crazy loop
			time.Sleep(5 * time.Second)
		}
	}
	utils.Logger.Info().Str("url", cam.ID).Str("action", "done").Msg("cam")
}

func (cam *Camera) runStreamOnce(ctx context.Context) error {
	var sourceUrl *url.URL
	var err error

	sourceUrl, err = cam.queryMediaUrl(ctx)
	if err != nil {
		return errors.Annotate(err, "queryMediaUrl")
	}

	utils.Logger.Info().Str("host", sourceUrl.Host).Msg("start!")
	if err = cam.rtspClient.Start(sourceUrl.Scheme, sourceUrl.Host); err != nil {
		return errors.Annotate(err, "start")
	}
	defer func() { _ = cam.rtspClient.Close() }()

	if options, err := cam.rtspClient.Options(sourceUrl); err != nil {
		return errors.Annotate(err, "options")
	} else {
		utils.Logger.Debug().Interface("options", options).Str("cam", cam.ID).Msg("camera")
	}
	tracks, trackUrl, _, err := cam.rtspClient.Describe(sourceUrl)
	if err != nil {
		return errors.Annotate(err, "describe")
	}

	sockRtp, err := push.NewSocket()
	if err != nil {
		return errors.Annotate(err, "push socket")
	}
	defer sockRtp.Close()
	if err = sockRtp.Dial(common.MakeSouthRtp(cam.ID)); err != nil {
		return errors.Annotate(err, "push connect")
	}
	sockRtp.SetOption(mangos.OptionNoDelay, true)

	sockRtcp, err := push.NewSocket()
	if err != nil {
		return errors.Annotate(err, "push socket")
	}
	defer sockRtcp.Close()
	if err = sockRtcp.Dial(common.MakeSouthRtp(cam.ID)); err != nil {
		return errors.Annotate(err, "push connect")
	}
	sockRtcp.SetOption(mangos.OptionNoDelay, true)

	cam.rtspClient.OnPacketRTP = func(ctx *gortsplib.ClientOnPacketRTPCtx) {
		b, err := ctx.Packet.Marshal()
		if err == nil {
			sockRtp.SetOption(mangos.OptionSendDeadline, -1)
			err = sockRtp.Send(b)
		}
		utils.Logger.Debug().
			Str("proto", "rtp").
			Int("track", ctx.TrackID).
			Uint16("seq", ctx.Packet.SequenceNumber).
			Str("z", ctx.Packet.String()).
			Err(err).
			Msg("stream")
	}

	cam.rtspClient.OnPacketRTCP = func(ctx *gortsplib.ClientOnPacketRTCPCtx) {
		b, err := ctx.Packet.Marshal()
		if err == nil {
			sockRtcp.SetOption(mangos.OptionSendDeadline, -1)
			err = sockRtcp.Send(b)
		}
		utils.Logger.Debug().
			Str("proto", "rtcp").
			Int("track", ctx.TrackID).
			Interface("z", ctx.Packet).
			Err(err).
			Msg("stream")
	}

	log.Printf("Tracks: %v", tracks)
	err = cam.rtspClient.SetupAndPlay(tracks[0:1], trackUrl)
	if err != nil {
		return errors.Annotate(err, "setupAndPlay")
	}

	<-ctx.Done()

	return nil
}

func (cam *Camera) queryMediaUrl(ctx context.Context) (*url.URL, error) {
	streamURI := cam.onvifClient.FetchStreamURI(ctx)
	sourceUrl, err := url.Parse(streamURI)
	if err != nil {
		return nil, errors.Annotate(err, "parse")
	}
	return sourceUrl, nil
}

func (cam *Camera) onCmdExit() {
	switch cam.State {
	case CamAgentOff:
		panic("BUG unexpected state")
	case CamAgentPlaying, CamAgentPausing, CamAgentResuming:
		cam.State = CamAgentPausing
		cam.group.Cancel()
		cam.group.Wait()
		cam.State = CamAgentIdle
		fallthrough
	case CamAgentIdle:
		return
	}
}

func (cam *Camera) onCmdPlay(ctx context.Context) {
	switch cam.State {
	case CamAgentOff:
		panic("BUG unexpected state")
	case CamAgentPausing, CamAgentResuming:
		cam.State = CamAgentResuming
		if cam.group.Count() <= 0 {
			return
		}
		cam.State = CamAgentIdle
		fallthrough
	case CamAgentIdle:
		cam.State = CamAgentPlaying
		cam.group = utils.NewGroup(ctx)
		cam.group.Run(func(c context.Context) { pipeline(c, common.MakeSouthRtp(cam.ID), common.MakeNorthRtp(cam.ID)) })
		cam.group.Run(func(c context.Context) { pipeline(c, common.MakeSouthRtcp(cam.ID), common.MakeNorthRtcp(cam.ID)) })
		cam.group.Run(func(c context.Context) { cam.runStream(c) })
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