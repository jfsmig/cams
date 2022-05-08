package main

import (
	"context"
	"github.com/aler9/gortsplib"
	"github.com/aler9/gortsplib/pkg/base"
	goonvif "github.com/use-go/onvif"
	"github.com/use-go/onvif/media"
	"github.com/use-go/onvif/xsd/onvif"
	"log"
	"strings"
	"time"
)

type OnVifDevice struct {
	generation uint32
	onvifURL   string
	dev        goonvif.Device
	client     gortsplib.Client
}

func (d *OnVifDevice) RunLoop(ctx context.Context, a *LanAgent) {
	var sourceUrl *base.URL
	var err error

	Logger.Info().Str("url", d.onvifURL).Str("action", "run").Msg("device")

	//d.dev.Authenticate(user, password)

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

	reply, err := call_GetStreamUri_parse_GetStreamUriResponse(d.dev, request)
	if err != nil {
		log.Panicln(err)
	} else {
		log.Println(reply)
	}

	sourceUrlRaw := strings.Replace(string(reply.MediaUri.Uri), "rtsp://", "rtsp://"+user+":"+password+"@", 1)
	sourceUrl, err = base.ParseURL(sourceUrlRaw)
	if err != nil {
		log.Panicln(err)
	} else {
		log.Printf("Stream URL: %v", sourceUrl)
	}

	if err = d.client.Start(sourceUrl.Scheme, sourceUrl.Host); err != nil {
		log.Panicln(err)
	}
	defer func() { _ = d.client.Close() }()

	if opts, err := d.client.Options(sourceUrl); err != nil {
		log.Panicln(err)
	} else {
		log.Printf("Options: %v", opts)
	}

	d.client.OnPacketRTP = func(c *gortsplib.Client) func(i int, bytes []byte) {
		return func(i int, bytes []byte) {
			log.Printf("RTP %d %d %d", i, len(bytes), cap(bytes))
		}
	}(&d.client)

	d.client.OnPacketRTCP = func(c *gortsplib.Client) func(i int, bytes []byte) {
		return func(i int, bytes []byte) {
			log.Printf("RTCP %d %d %d", i, len(bytes), cap(bytes))
		}
	}(&d.client)

	if tracks, trackUrl, _, err := d.client.Describe(sourceUrl); err != nil {
		log.Panicln(err)
	} else {
		log.Printf("Tracks: %v", tracks)
		err := d.client.SetupAndPlay(tracks, trackUrl)
		if err != nil {
			log.Panicln(err)
		}
	}
	time.Sleep(5 * time.Second)

	if resp, err := d.client.Pause(); err != nil {
		log.Panicln(err)
	} else {
		log.Printf("Pause: %v", resp)
	}
}

func (dev *OnVifDevice) Shut() {

}

//go:generate go run github.com/jfsmig/wiy/cmd/gen-parse GetStreamUriResponse_auto.go main media.GetStreamUri media.GetStreamUriResponse
