// Copyright (c) 2022-2024 The authors (see the AUTHORS file)
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package camera

import (
	"context"
	"sync"
	"time"

	"github.com/jfsmig/cams/go/rtsp1"
	"github.com/jfsmig/cams/go/rtsp1/pkg/media"
	"github.com/jfsmig/cams/go/rtsp1/pkg/url"
	"github.com/jfsmig/cams/go/transport"
	"github.com/jfsmig/cams/go/utils"
	"github.com/jfsmig/onvif/sdk"
	"github.com/juju/errors"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
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
	rtspClient  rtsp1.Client

	requests chan CamCommand

	group utils.Swarm

	flagRetry bool
}

func NewCamera(open UploadOpenFunc, appliance sdk.Appliance) *Camera {
	return &Camera{
		open:        open,
		ID:          appliance.GetUUID(),
		generation:  0,
		onvifClient: appliance,
		rtspClient: rtsp1.Client{
			ReadTimeout:     5 * time.Second,
			WriteTimeout:    5 * time.Second,
			RedirectDisable: true,
			AnyPortEnable:   true,
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

	// Prepare the camera RTSP side
	sourceUrl, err = cam.queryMediaUrl(ctx)
	if err != nil {
		return errors.Annotate(err, "queryMediaUrl")
	}

	cam.debug().Str("url", sourceUrl.Host).Msg("cam streaming")

	if err = cam.rtspClient.Start(ctx, sourceUrl.Scheme, sourceUrl.Host); err != nil {
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

	// Prepare the camera RTP/RTCP side
	udpListener := transport.NewRawUdpListener()
	if err := udpListener.OpenPair("0.0.0.0"); err != nil {
		utils.Logger.Panic().Err(err).Msg("udp listener error")
	}
	defer udpListener.Close()

	for _, m := range medias {
		if m.Type != media.TypeVideo {
			continue
		}
		_, err = cam.rtspClient.Setup(m, baseUrl, udpListener.GetPortMedia(), udpListener.GetPortControl())
		if err != nil {
			return errors.Annotate(err, "RTSP Setup")
		} else {
			utils.Logger.Info().Interface("media", *m).Msg("RTSP Setup")
		}
	}

	if err = upload.OnSDP(sdp); err != nil {
		return errors.Annotate(err, "send sdp banner")
	}

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return udpListener.Run(ctx)
	})
	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case pkt := <-udpListener.GetMediaChannel():
				decoded := rtp.Header{}
				if _, err := decoded.Unmarshal(pkt); err != nil {
					utils.Logger.Warn().Int("size", len(pkt)).Err(err).Msg("rtp")
				} else {
					if err = upload.OnRTP(pkt); err != nil {
						return err
					}
				}
			case pkt := <-udpListener.GetControlChannel():
				decoded := rtcp.Header{}
				if err := decoded.Unmarshal(pkt); err != nil {
					utils.Logger.Warn().Int("size", len(pkt)).Err(err).Msg("rtcp")
				} else {
					if err = upload.OnRTCP(pkt); err != nil {
						return err
					}
				}
			}
		}
	})

	// Spawn goroutines that will consume the camera stream
	_, err = cam.rtspClient.Play(nil)
	defer cam.rtspClient.Pause()
	if err != nil {
		return errors.Annotate(err, "play")
	}

	return g.Wait()
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
