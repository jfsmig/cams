package main

import (
	"archive/tar"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jfsmig/cams/go/camera"
	"github.com/jfsmig/cams/go/utils"
	"github.com/jfsmig/onvif/networking"
	"github.com/jfsmig/onvif/sdk"
	wsdiscovery "github.com/jfsmig/onvif/ws-discovery"
	"github.com/juju/errors"
)

var authInfo = networking.ClientAuth{
	"admin",
	"ollyhgqo",
}

func camPlay(ctx context.Context, addr string) error {
	// FIXME(jfsmig): We currently need a UUID that is only provided by a discovery. it sucks as is.
	allClientInfo, err := wsdiscovery.GetAvailableDevicesAtSpecificEthernetInterface("enp5s0")
	if err != nil {
		return errors.Annotate(err, "discover")
	}

	for _, clientInfo := range allClientInfo {
		if clientInfo.Xaddr != addr {
			continue
		}

		dev, err := sdk.NewDevice(ctx, clientInfo, authInfo, http.DefaultClient)
		if err != nil {
			return errors.Annotate(err, "OnVif new error")
		}
		utils.Logger.Info().Interface("device", clientInfo).Msg("OnVif device ready")

		cam := camera.NewCamera(NewLocalUploadMaker(), dev)
		utils.Logger.Info().Interface("camera", cam).Msg("camera ready")

		cam.NoRetry()

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			cam.Run(ctx)
		}()

		cam.PlayStream()
		utils.Logger.Info().Str("action", "start").Msg("stream command")

		select {
		case <-ctx.Done():
		case <-time.After(10 * time.Second):
		}
		utils.Logger.Info().Msg("stream ending")

		cam.StopStream()
		utils.Logger.Info().Str("action", "stop").Msg("stream command")

		cam.Exit()
		wg.Wait()
		return nil
	}

	return errors.New("camera not found")
}

func NewLocalUploadMaker() camera.UploadOpenFunc {
	return func(ctx context.Context) (camera.UpstreamMedia, error) {
		fout, err := ioutil.TempFile("", "cams-capture-*.tar")
		if err != nil {
			return nil, errors.Annotate(err, "mktemp")
		}
		w := tar.NewWriter(fout)
		if err != nil {
			return nil, errors.Annotate(err, "create")
		}
		return &localUpstream{
			file:    fout,
			archive: w,
		}, nil
	}
}

type localUpstream struct {
	file          *os.File
	archive       *tar.Writer
	packetCounter atomic.Uint64
}

func (lu *localUpstream) Close() {
	lu.archive.Flush()
	lu.archive.Close()
	lu.file.Close()
}

func (lu *localUpstream) OnSDP(sdp string) error {
	return lu.writeFile("sdp", []byte(sdp))
}

func (lu *localUpstream) OnRTP(pkt []byte) error {
	return lu.writeFile("rtp", pkt)
}

func (lu *localUpstream) OnRTCP(pkt []byte) error {
	return lu.writeFile("rtcp", pkt)
}

func (lu *localUpstream) writeFile(tag string, payload []byte) error {
	idx := lu.packetCounter.Add(1)
	path := fmt.Sprintf("%06d", idx) + "." + tag
	sz := int64(len(payload))
	utils.Logger.Info().Str("path", path).Int64("size", sz).Msg("entry")
	hdr := tar.Header{
		Name:       path,
		Size:       sz,
		AccessTime: time.Now(),
		ModTime:    time.Now(),
		ChangeTime: time.Now(),
		Mode:       0644,
		Typeflag:   tar.TypeReg,
		Format:     tar.FormatGNU,
	}
	if err := lu.archive.WriteHeader(&hdr); err != nil {
		return errors.Annotate(err, "tar header")
	}
	_, err := lu.archive.Write(payload)
	return errors.Annotate(err, "tar body")
}
