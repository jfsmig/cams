package main

import (
	"context"
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jfsmig/streaming/rtsp1"
	"github.com/jfsmig/streaming/rtsp1/pkg/url"
	"github.com/jfsmig/streaming/transport"
	"github.com/pion/rtp"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

var (
	Username string = envOrDefault("ONVIF_USERNAME", "admin")
	Password string = envOrDefault("ONVIF_PASSWORD", "admin")
)

var (
	Logger = zerolog.
		New(zerolog.ConsoleWriter{
			Out: os.Stderr, TimeFormat: time.RFC3339,
		}).With().Timestamp().Logger()
)

func envOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	} else {
		return defaultValue
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	flag.Parse()
	uStr := flag.Arg(0)
	uStr = strings.Replace(uStr, "rtsp://", "rtsp://"+Username+":"+Password+"@", 1)
	u, err := url.Parse(uStr)
	if err != nil {
		Logger.Panic().Err(err).Msg("url parsing error")
	}

	client := rtsp1.Client{
		AnyPortEnable: true,
	}
	defer func() {
		if e := client.Close(); e != nil {
			Logger.Warn().Err(e).Msg("Close")
		} else {
			Logger.Info().Msg("Close")
		}
	}()

	err = client.Start(ctx, u.Scheme, u.Host)
	if err != nil {
		Logger.Panic().Err(err).Msg("Start error")
	}

	medias, _, reply, err := client.Describe(u)
	if err != nil {
		Logger.Panic().Err(err).Msg("Describe error")
	} else {
		Logger.Info().Interface("medias", medias).Msg("Describe")
		log.Println(string(reply.Body))
	}

	// Prepare the background handling of the RTP & RTCP frames
	udpListener := transport.UdpListener{}
	if err := udpListener.OpenPair("0.0.0.0"); err != nil {
		Logger.Panic().Err(err).Msg("udp listener error")
	}
	defer udpListener.Close()
	outMedia := make(chan []byte, 64)
	outControl := make(chan []byte, 4)
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return udpListener.Run(ctx, outMedia, outControl)
	})
	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case pkt := <-outMedia:
				decoded := rtp.Header{}
				if _, err := decoded.Unmarshal(pkt); err != nil {
					Logger.Warn().Int("size", len(pkt)).Err(err).Msg("rtp")
				} else {
					Logger.Info().Int("size", len(pkt)).Interface("pkt", decoded).Msg("rtp")
				}
			case pkt := <-outControl:
				Logger.Info().Int("size", len(pkt)).Msg("rtcp")
			}
		}
	})

	//mediaVideo := medias.FindFormat(&format.H264{})
	//mediaAudio := medias.FindFormat(&format.G711{})
	for _, media := range medias {
		_, err = client.Setup(media, u, udpListener.GetPortMedia(), udpListener.GetPortControl())
		if err != nil {
			Logger.Panic().Err(err).Interface("media", *media).Msg("Setup error")
		} else {
			Logger.Info().Interface("media", *media).Msg("Setup")
		}
	}

	_, err = client.Play(nil)
	if err != nil {
		Logger.Panic().Err(err).Msg("Play error")
	} else {
		Logger.Info().Msg("Play")
	}

	// Arm a timer and wait for the end of the main loop
	g.Go(func() error {
		time.Sleep(10 * time.Second)
		cancel()
		return nil
	})
	if err := g.Wait(); err != nil {
		Logger.Error().Msg("main loop error")
	}
}
