// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"github.com/use-go/onvif/device"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/aler9/gortsplib"
	"github.com/aler9/gortsplib/pkg/url"
	"github.com/juju/errors"
	goonvif "github.com/use-go/onvif"
	"github.com/use-go/onvif/media"
	sdev "github.com/use-go/onvif/sdk/device"
	smedia "github.com/use-go/onvif/sdk/media"
	"github.com/use-go/onvif/xsd/onvif"
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
type LanCamera struct {
	ID string

	singletonLock sync.Mutex
	State         CamAgentState

	endpoint string
	user     string
	password string

	generation uint32

	onvifClient *goonvif.Device
	rtspClient  gortsplib.Client

	requests chan CamCommand

	group utils.Swarm
}

func NewCamera(id, endpoint string) (*LanCamera, error) {
	authenticatedDevice, err := goonvif.NewDevice(goonvif.DeviceParams{
		Xaddr:    endpoint,
		Username: user,
		Password: password,
	})
	if err != nil {
		return nil, errors.Annotate(err, "authentication failed")
	}
	transport := gortsplib.TransportUDP

	return &LanCamera{
		ID:          id,
		endpoint:    endpoint,
		user:        user,
		password:    password,
		generation:  0,
		onvifClient: authenticatedDevice,
		rtspClient: gortsplib.Client{
			ReadTimeout:           5 * time.Second,
			WriteTimeout:          5 * time.Second,
			RedirectDisable:       true,
			AnyPortEnable:         true,
			Transport:             &transport,
			InitialUDPReadTimeout: 3 * time.Second,
		},
		requests: make(chan CamCommand, 1),
	}, nil
}

func runCam(cam *LanCamera) utils.SwarmFunc {
	return func(ctx context.Context) { cam.Run(ctx) }
}

func (cam *LanCamera) Run(ctx context.Context) {
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
		cam.onvifClient = nil
	}()

	authenticatedDevice, err := goonvif.NewDevice(goonvif.DeviceParams{
		Xaddr:    cam.endpoint,
		Username: user,
		Password: password,
	})
	if err != nil {
		utils.Logger.Error().Err(err).Str("action", "auth").Msg("cam")
		return
	}
	transport := gortsplib.TransportUDP
	cam.onvifClient = authenticatedDevice
	cam.rtspClient.Transport = &transport

	if reply, err := sdev.Call_GetSystemUris(ctx, cam.onvifClient, device.GetSystemUris{}); err == nil {
		utils.Logger.Info().Interface("uris", reply).Msg("cam")
	} else {
		utils.Logger.Info().Err(err).Interface("uris", nil).Msg("cam")
	}
	if reply, err := sdev.Call_GetEndpointReference(ctx, cam.onvifClient, device.GetEndpointReference{}); err == nil {
		utils.Logger.Info().Interface("ep_ref", reply).Msg("cam")
	} else {
		utils.Logger.Info().Err(err).Interface("ep_ref", nil).Msg("cam")
	}
	if reply, err := sdev.Call_GetEndpointReference(ctx, cam.onvifClient, device.GetEndpointReference{}); err == nil {
		utils.Logger.Info().Interface("ep_ref", reply).Msg("cam")
	} else {
		utils.Logger.Info().Err(err).Interface("ep_ref", nil).Msg("cam")
	}
	if reply, err := sdev.Call_GetServices(ctx, cam.onvifClient, device.GetServices{}); err == nil {
		utils.Logger.Info().Interface("srv", reply).Msg("cam")
	} else {
		utils.Logger.Info().Err(err).Interface("srv", nil).Msg("cam")
	}
	if reply, err := sdev.Call_GetServiceCapabilities(ctx, cam.onvifClient, device.GetServiceCapabilities{}); err == nil {
		utils.Logger.Info().Interface("capabilities", reply).Msg("cam")
	} else {
		utils.Logger.Info().Err(err).Interface("capabilities", nil).Msg("cam")
	}

	if reply, err := smedia.Call_GetVideoSources(ctx, cam.onvifClient, media.GetVideoSources{}); err == nil {
		utils.Logger.Info().Interface("video_sources", reply).Msg("cam")
	} else {
		utils.Logger.Info().Err(err).Interface("video_source", nil).Msg("cam")
	}
	if reply, err := smedia.Call_GetVideoSourceConfigurations(ctx, cam.onvifClient, media.GetVideoSourceConfigurations{}); err == nil {
		utils.Logger.Info().Interface("video_source_config", reply).Msg("cam")
	} else {
		utils.Logger.Info().Err(err).Interface("video_source_config", nil).Msg("cam")
	}
	if reply, err := smedia.Call_GetVideoSourceModes(ctx, cam.onvifClient, media.GetVideoSourceModes{}); err == nil {
		utils.Logger.Info().Interface("video_source_modes", reply).Msg("cam")
	} else {
		utils.Logger.Info().Err(err).Interface("video_source_modes", nil).Msg("cam")
	}
	if reply, err := smedia.Call_GetVideoAnalyticsConfiguration(ctx, cam.onvifClient, media.GetVideoAnalyticsConfiguration{}); err == nil {
		utils.Logger.Info().Interface("video_analytics_config", reply).Msg("cam")
	} else {
		utils.Logger.Info().Err(err).Interface("video_analytics_config", nil).Msg("cam")
	}
	if reply, err := smedia.Call_GetVideoEncoderConfiguration(ctx, cam.onvifClient, media.GetVideoEncoderConfiguration{}); err == nil {
		utils.Logger.Info().Interface("video_encoder_config", reply).Msg("cam")
	} else {
		utils.Logger.Info().Err(err).Interface("video_encoder_config", nil).Msg("cam")
	}
	if reply, err := smedia.Call_GetVideoEncoderConfigurations(ctx, cam.onvifClient, media.GetVideoEncoderConfigurations{}); err == nil {
		utils.Logger.Info().Interface("video_encoder_configs", reply).Msg("cam")
	} else {
		utils.Logger.Info().Err(err).Interface("video_encoder_configs", nil).Msg("cam")
	}

	if reply, err := smedia.Call_GetAudioSources(ctx, cam.onvifClient, media.GetAudioSources{}); err == nil {
		utils.Logger.Info().Interface("audio", reply).Msg("cam")
	}
	if reply, err := smedia.Call_GetAudioDecoderConfigurations(ctx, cam.onvifClient, media.GetAudioDecoderConfigurations{}); err == nil {
		utils.Logger.Info().Interface("audio_decoder_config", reply).Msg("cam")
	}
	if reply, err := smedia.Call_GetAudioSourceConfigurations(ctx, cam.onvifClient, media.GetAudioSourceConfigurations{}); err == nil {
		utils.Logger.Info().Interface("audio_source_config", reply).Msg("cam")
	}
	if reply, err := smedia.Call_GetMetadataConfigurations(ctx, cam.onvifClient, media.GetMetadataConfigurations{}); err == nil {
		utils.Logger.Info().Interface("metadata_config", reply).Msg("cam")
	}

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

func (cam *LanCamera) PK() string  { return cam.ID }
func (cam *LanCamera) Ping()       { cam.requests <- CamCommandPing }
func (cam *LanCamera) Exit()       { cam.requests <- CamCommandExit }
func (cam *LanCamera) PlayStream() { cam.requests <- CamCommandPlay }
func (cam *LanCamera) StopStream() { cam.requests <- CamCommandPause }

func (cam *LanCamera) runStream(ctx context.Context) {
	for ctx.Err() == nil {
		utils.Logger.Debug().Str("url", cam.endpoint).Str("action", "start").Msg("cam")
		err := cam.runStreamOnce(ctx)
		if err != nil {
			utils.Logger.Warn().Str("url", cam.endpoint).Str("action", "done").Err(err).Msg("cam")
		} else {
			// Avoid a crazy loop
			time.Sleep(5 * time.Second)
		}
	}
	utils.Logger.Info().Str("url", cam.endpoint).Str("action", "done").Msg("cam")
}

func (cam *LanCamera) runStreamOnce(ctx context.Context) error {
	var sourceUrl *url.URL
	var err error

	sourceUrl, err = cam.queryMediaUrl(ctx)
	if err != nil {
		return errors.Annotate(err, "getMediaUrl")
	}

	utils.Logger.Info().Str("host", sourceUrl.Host).Msg("start!")
	if err = cam.rtspClient.Start(sourceUrl.Scheme, sourceUrl.Host); err != nil {
		return errors.Annotate(err, "start")
	}
	defer func() { _ = cam.rtspClient.Close() }()

	if _, err := cam.rtspClient.Options(sourceUrl); err != nil {
		return errors.Annotate(err, "options")
	}

	sockRtp, err := push.NewSocket()
	if err != nil {
		return errors.Annotate(err, "push socket")
	}
	defer sockRtp.Close()
	if err = sockRtp.Dial(makeSouthRtp(cam.ID)); err != nil {
		return errors.Annotate(err, "push connect")
	}

	sockRtcp, err := push.NewSocket()
	if err != nil {
		return errors.Annotate(err, "push socket")
	}
	defer sockRtcp.Close()
	if err = sockRtcp.Dial(makeSouthRtp(cam.ID)); err != nil {
		return errors.Annotate(err, "push connect")
	}

	cam.rtspClient.OnPacketRTP = func(ctx *gortsplib.ClientOnPacketRTPCtx) {
		b, err := ctx.Packet.Marshal()
		if err != nil {
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
		if err != nil {
			err = sockRtcp.Send(b)
		}
		utils.Logger.Debug().
			Str("proto", "rtcp").
			Int("track", ctx.TrackID).
			Interface("z", ctx.Packet).
			Err(err).
			Msg("stream")
	}

	tracks, trackUrl, _, err := cam.rtspClient.Describe(sourceUrl)
	if err != nil {
		return errors.Annotate(err, "describe")
	}

	log.Printf("Tracks: %v", tracks)
	err = cam.rtspClient.SetupAndPlay(tracks, trackUrl)
	if err != nil {
		return errors.Annotate(err, "setupAndPlay")
	}

	time.Sleep(5 * time.Second)

	if _, err := cam.rtspClient.Pause(); err != nil {
		return errors.Annotate(err, "pause")
	}

	return nil
}

func (cam *LanCamera) queryMediaUrl(ctx context.Context) (*url.URL, error) {
	request := media.GetStreamUri{
		StreamSetup: onvif.StreamSetup{
			Stream: onvif.StreamType("000"),
			Transport: onvif.Transport{
				Protocol: onvif.TransportProtocol("RTSP"),
				Tunnel:   nil,
			},
		},
		ProfileToken: onvif.ReferenceToken("000"),
	}

	mediaUriReply, err := smedia.Call_GetStreamUri(ctx, cam.onvifClient, request)
	if err != nil {
		return nil, errors.Annotate(err, "rpc")
	}

	sourceUrlRaw := strings.Replace(string(mediaUriReply.MediaUri.Uri),
		"rtsp://", "rtsp://"+cam.user+":"+cam.password+"@", 1)

	sourceUrl, err := url.Parse(sourceUrlRaw)
	if err != nil {
		return nil, errors.Annotate(err, "parse")
	}
	return sourceUrl, nil
}

func (cam *LanCamera) onCmdExit() {
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

func (cam *LanCamera) onCmdPlay(ctx context.Context) {
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
		cam.group.Run(func(c context.Context) { pipeline(c, makeSouthRtp(cam.ID), makeNorthRtp(cam.ID)) })
		cam.group.Run(func(c context.Context) { pipeline(c, makeSouthRtcp(cam.ID), makeNorthRtcp(cam.ID)) })
		cam.group.Run(func(c context.Context) { cam.runStream(c) })
		fallthrough
	case CamAgentPlaying:
		// No-Op
	default:
		panic("BUG invalid state")
	}
}

func (cam *LanCamera) onCmdStop() {
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

func (cam *LanCamera) onCmdPing(ctx context.Context) {
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
