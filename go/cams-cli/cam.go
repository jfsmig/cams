package main

import (
	"context"
	"fmt"
	"github.com/aler9/gortsplib/v2/pkg/format"
	"github.com/aler9/gortsplib/v2/pkg/media"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"net/http"
	"os"
	"path/filepath"
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

		tmpDirPath, err := os.MkdirTemp("", "cams-capture-")
		if err != nil {
			return errors.Annotate(err, "mktemp")
		}
		utils.Logger.Info().Str("path", tmpDirPath).Msg("Temporary directory path ready")

		cam := camera.NewCamera(NewLocalUploadMaker(tmpDirPath), dev)
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

type localUpstream struct {
	baseDirectoryPath string
	packetCounter     atomic.Uint64
}

func (lu *localUpstream) Close() {}

func (lu *localUpstream) OnSDP(sdp []byte) error {
	return lu.writeFile("sdp", sdp)
}

func (lu *localUpstream) OnRTP(m *media.Media, f format.Format, pkt *rtp.Packet) error {
	payload, err := pkt.Marshal()
	if err != nil {
		return errors.Annotate(err, "rtp marshalling")
	}
	return lu.writeFile("rtp", payload)
}

func (lu *localUpstream) OnRTCP(m *media.Media, pkt *rtcp.Packet) error {
	payload, err := (*pkt).Marshal()
	if err != nil {
		return errors.Annotate(err, "rtcp marshalling")
	}
	return lu.writeFile("rtcp", payload)
}

func (lu *localUpstream) writeFile(tag string, payload []byte) error {
	idx := lu.packetCounter.Add(1)
	basename := fmt.Sprintf("%06d", idx) + "." + tag
	fout, err := os.OpenFile(filepath.Join(lu.baseDirectoryPath, basename), os.O_EXCL|os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return errors.Annotatef(err, "open error on [%v]", basename)
	}
	defer fout.Close()
	_, err = fout.Write(payload)
	return errors.Annotatef(err, "write error on [%v]", basename)
}

func NewLocalUploadMaker(path string) camera.UploadOpenFunc {
	return func(ctx context.Context) (camera.UpstreamMedia, error) {
		return &localUpstream{
			baseDirectoryPath: path,
		}, nil
	}
}
