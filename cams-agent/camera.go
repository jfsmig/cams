// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"github.com/aler9/gortsplib"
	"github.com/aler9/gortsplib/pkg/url"
	"github.com/jfsmig/cams/utils"
	"github.com/juju/errors"
	goonvif "github.com/use-go/onvif"
	"github.com/use-go/onvif/media"
	sdk "github.com/use-go/onvif/sdk/media"
	"github.com/use-go/onvif/xsd/onvif"
	"log"
	"strings"
	"time"
)

// We assume only one stream per camera.
type LanCamera struct {
	ID string

	endpoint string
	user     string
	password string

	generation uint32

	onvifClient *goonvif.Device
	rtspClient  gortsplib.Client
}

func (s *LanCamera) PK() string { return s.ID }

func (d *LanCamera) GetMediaUrl(ctx context.Context) (*url.URL, error) {
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

	mediaUriReply, err := sdk.Call_GetStreamUri(ctx, d.onvifClient, request)
	if err != nil {
		return nil, errors.Annotate(err, "rpc")
	}

	sourceUrlRaw := strings.Replace(string(mediaUriReply.MediaUri.Uri),
		"rtsp://", "rtsp://"+d.user+":"+d.password+"@", 1)

	sourceUrl, err := url.Parse(sourceUrlRaw)
	if err != nil {
		return nil, errors.Annotate(err, "parse")
	}
	return sourceUrl, nil
}

func (d *LanCamera) PlayStream(ctx context.Context, a *LanAgent) error {
	var sourceUrl *url.URL
	var err error

	sourceUrl, err = d.GetMediaUrl(ctx)
	if err != nil {
		return errors.Annotate(err, "getMediaUrl")
	}

	utils.Logger.Info().Str("host", sourceUrl.Host).Msg("start!")
	if err = d.rtspClient.Start(sourceUrl.Scheme, sourceUrl.Host); err != nil {
		return errors.Annotate(err, "start")
	}
	defer func() { _ = d.rtspClient.Close() }()

	if _, err := d.rtspClient.Options(sourceUrl); err != nil {
		return errors.Annotate(err, "options")
	}

	d.rtspClient.OnPacketRTP = func(ctx *gortsplib.ClientOnPacketRTPCtx) {
		utils.Logger.Debug().Str("proto", "rtp").Int("track", ctx.TrackID).Uint16("seq", ctx.Packet.SequenceNumber).Str("z", ctx.Packet.String()).Msg("stream")
	}

	d.rtspClient.OnPacketRTCP = func(ctx *gortsplib.ClientOnPacketRTCPCtx) {
		utils.Logger.Debug().Str("proto", "rtcp").Int("track", ctx.TrackID).Interface("z", ctx.Packet).Msg("stream")
	}

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

func (d *LanCamera) RunLoop(ctx context.Context, a *LanAgent) {
	utils.Logger.Debug().Str("url", d.endpoint).Str("action", "run").Msg("device")
	err := d.PlayStream(ctx, a)
	if err != nil {
		utils.Logger.Warn().Str("url", d.endpoint).Str("action", "done").Err(err).Msg("device")
	} else {
		utils.Logger.Info().Str("url", d.endpoint).Str("action", "done").Msg("device")
	}
}

func (d *LanCamera) StopStream() {

}