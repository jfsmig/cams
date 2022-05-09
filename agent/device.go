package main

import (
	"context"
	"github.com/aler9/gortsplib"
	"github.com/aler9/gortsplib/pkg/base"
	"github.com/juju/errors"
	goonvif "github.com/use-go/onvif"
	"github.com/use-go/onvif/media"
	"github.com/use-go/onvif/xsd/onvif"
	"log"
	"strings"
	"time"
)

type OnVifDevice struct {
	endpoint string
	user     string
	password string

	generation uint32

	onvifClient *goonvif.Device
	rtspClient  gortsplib.Client
}

func (d *OnVifDevice) GetMediaUrl(ctx context.Context) (*base.URL, error) {
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

	mediaUriReply, err := call_GetStreamUri_parse_GetStreamUriResponse(d.onvifClient, request)
	if err != nil {
		return nil, errors.Annotate(err, "rpc")
	}

	sourceUrlRaw := strings.Replace(string(mediaUriReply.MediaUri.Uri),
		"rtsp://", "rtsp://"+d.user+":"+d.password+"@", 1)

	sourceUrl, err := base.ParseURL(sourceUrlRaw)
	if err != nil {
		return nil, errors.Annotate(err, "parse")
	}
	return sourceUrl, nil
}

func (d *OnVifDevice) ConsumeStream(ctx context.Context, a *LanAgent) error {
	var sourceUrl *base.URL
	var err error

	sourceUrl, err = d.GetMediaUrl(ctx)
	if err != nil {
		return errors.Annotate(err, "getMediaUrl")
	}

	Logger.Info().Str("host", sourceUrl.Host).Msg("start!")
	if err = d.rtspClient.Start(sourceUrl.Scheme, sourceUrl.Host); err != nil {
		return errors.Annotate(err, "start")
	}
	defer func() { _ = d.rtspClient.Close() }()

	if _, err := d.rtspClient.Options(sourceUrl); err != nil {
		return errors.Annotate(err, "options")
	}

	d.rtspClient.OnPacketRTP = func(c *gortsplib.Client) func(i int, bytes []byte) {
		return func(i int, bytes []byte) {
			log.Printf("RTP %d %d %d", i, len(bytes), cap(bytes))
		}
	}(&d.rtspClient)

	d.rtspClient.OnPacketRTCP = func(c *gortsplib.Client) func(i int, bytes []byte) {
		return func(i int, bytes []byte) {
			log.Printf("RTCP %d %d %d", i, len(bytes), cap(bytes))
		}
	}(&d.rtspClient)

	tracks, trackUrl, _, err := d.rtspClient.Describe(sourceUrl)
	if err != nil {
		return errors.Annotate(err, "describe")
	}

	log.Printf("Tracks: %v", tracks)
	err = d.rtspClient.SetupAndPlay(tracks, trackUrl)
	if err != nil {
		return errors.Annotate(err, "setupAndPlay")
	}

	time.Sleep(5 * time.Second)

	if _, err := d.rtspClient.Pause(); err != nil {
		return errors.Annotate(err, "pause")
	}

	return nil
}

func (d *OnVifDevice) RunLoop(ctx context.Context, a *LanAgent) {
	Logger.Info().Str("url", d.endpoint).Str("action", "run").Msg("device")
	err := d.ConsumeStream(ctx, a)
	if err != nil {
		Logger.Warn().Str("url", d.endpoint).Str("action", "done").Err(err).Msg("device")
	} else {
		Logger.Info().Str("url", d.endpoint).Str("action", "done").Msg("device")
	}
}

func (d *OnVifDevice) Shut() {

}

//go:generate go run github.com/jfsmig/wiy/cmd/gen-parse GetStreamUriResponse_auto.go main media.GetStreamUri media.GetStreamUriResponse
