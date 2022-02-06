package main

import (
	"github.com/aler9/gortsplib"
	"github.com/aler9/gortsplib/pkg/base"
	goonvif "github.com/use-go/onvif"
	"github.com/use-go/onvif/media"
	"github.com/use-go/onvif/xsd/onvif"
	"log"
	"strings"
	"time"
)

var (
	user         = "admin"
	password     = "ollyhgqo"
	netInterface = "enp5s0"
)

func main() {
	devices := goonvif.GetAvailableDevicesAtSpecificEthernetInterface(netInterface)
	for _, dev := range devices {

		log.Printf("device %s", dev)
		for k, v := range dev.GetServices() {
			log.Printf("service %s %s", k, v)
		}

		dev.Authenticate(user, password)

		var sourceUrl *base.URL
		var err error

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

		reply, err := call_GetStreamUri_parse_GetStreamUriResponse(dev, request)
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

		transport := gortsplib.TransportUDP
		rtspClient := gortsplib.Client{
			ReadTimeout:           5 * time.Second,
			WriteTimeout:          5 * time.Second,
			RedirectDisable:       true,
			AnyPortEnable:         true,
			Transport:             &transport,
			InitialUDPReadTimeout: 3 * time.Second,
		}
		if err = rtspClient.Start(sourceUrl.Scheme, sourceUrl.Host); err != nil {
			log.Panicln(err)
		}
		defer func() { _ = rtspClient.Close() }()

		if opts, err := rtspClient.Options(sourceUrl); err != nil {
			log.Panicln(err)
		} else {
			log.Printf("Options: %v", opts)
		}

		if tracks, _, _, err := rtspClient.Describe(sourceUrl); err != nil {
			log.Panicln(err)
		} else {
			log.Printf("Tracks: %v", tracks)
		}
	}
}

//go:generate go run github.com/jfsmig/wiy/cmd/gen-parse GetStreamUriResponse_auto.go main media.GetStreamUri media.GetStreamUriResponse
