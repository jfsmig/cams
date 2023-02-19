// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package camera

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aler9/gortsplib/v2"
	"github.com/aler9/gortsplib/v2/pkg/format"
	"github.com/aler9/gortsplib/v2/pkg/media"
	"github.com/aler9/gortsplib/v2/pkg/url"
	"github.com/jfsmig/cams/go/utils"
	"github.com/jfsmig/onvif/sdk"
	"github.com/juju/errors"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/rs/zerolog"
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
	CamCommandExit
	CamCommandPlay
	CamCommandPause
)

// We assume only one stream per camera.
type Camera struct {
	open UploadOpenFunc

	ID            string
	generation    uint32
	singletonLock sync.Mutex
	State         CamAgentState

	onvifClient sdk.Appliance
	rtspClient  gortsplib.Client

	requests chan CamCommand

	group utils.Swarm

	flagRetry bool
}

func NewCamera(open UploadOpenFunc, appliance sdk.Appliance) *Camera {
	transport := gortsplib.TransportUDP
	return &Camera{
		open:        open,
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
		requests:  make(chan CamCommand, 8),
		flagRetry: true,
	}
}

func (cam *Camera) NoRetry() { cam.flagRetry = false }

func (cam *Camera) GetGeneration() uint32 { return cam.generation }

func (cam *Camera) SetGeneration(gen uint32) { cam.generation = gen }

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
			case CamCommandPing:
				cam.onCmdPing(ctx)
			case CamCommandExit:
				cam.group.Cancel()
				cam.group.Wait()
				return
			case CamCommandPlay:
				cam.onCmdPlay(ctx)
			case CamCommandPause:
				cam.onCmdStop()
			}
		}
	}
}

func (cam *Camera) PK() string  { return cam.ID }
func (cam *Camera) Ping()       { cam.requests <- CamCommandPing }
func (cam *Camera) Exit()       { cam.requests <- CamCommandExit }
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
		if !cam.flagRetry {
			break
		}
	}
	cam.debug().Str("url", cam.ID).Msg("cam stream exiting")
}

func (cam *Camera) runStreamOnce(ctx context.Context) error {
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

	medias, baseUrl, sdpResp, err := cam.rtspClient.Describe(sourceUrl)
	if err != nil {
		return errors.Annotate(err, "describe")
	}
	sdp := string(sdpResp.Body)
	cam.debug().
		Str("url", baseUrl.String()).
		Str("sdp", sdp).
		Interface("medias", medias).
		Msg("streams described")

	// Prepare the upstream side
	upload, err := cam.open(ctx)
	if err != nil {
		return errors.Annotate(err, "dial")
	}
	defer upload.Close()

	var fmt *format.H264
	mediaH264 := medias.FindFormat(&fmt)
	if mediaH264 == nil {
		return errors.New("no h264")
	} else {
		_, err = cam.rtspClient.Setup(mediaH264, baseUrl, 0, 0)
		if err != nil {
			return errors.Annotate(err, "setup")
		}
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
	activityJiffies := atomic.Uint64{}
	activityCheckTicker := time.Tick(3 * time.Second)

	cam.rtspClient.OnPacketRTPAny(func(m *media.Media, f format.Format, pkt *rtp.Packet) {
		activityJiffies.Add(1)
		if err2 := upload.OnRTP(m, f, pkt); err2 != nil {
			cam.warn(err2).Msg("rtp upload")
			if !localStop.Swap(true) {
				localError <- err2
			}
		}
	})

	cam.rtspClient.OnPacketRTCPAny(func(m *media.Media, pkt rtcp.Packet) {
		activityJiffies.Add(1)
		if err2 := upload.OnRTCP(m, &pkt); err2 != nil {
			cam.warn(err2).Msg("rtcp upload")
			if !localStop.Swap(true) {
				localError <- err2
			}
		}
	})

	if err = upload.OnSDP(sdp); err != nil {
		return errors.Annotate(err, "send sdp banner")
	}

	// Spawn goroutines that will consume the camera stream
	_, err = cam.rtspClient.Play(nil)
	defer cam.rtspClient.Pause()
	if err != nil {
		return errors.Annotate(err, "play")
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
	cam.debug().Str("source", sourceUrl.String()).Str("parsed", streamURI).Msg("STREAM")
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
