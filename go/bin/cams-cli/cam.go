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

package main

import (
	"archive/tar"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/jfsmig/cams/go/camagent"
	"github.com/jfsmig/cams/go/camctrl"
	"github.com/jfsmig/cams/go/mediabus"
	"github.com/jfsmig/cams/go/utils"
	"github.com/jfsmig/onvif/networking"
	"github.com/jfsmig/onvif/sdk"
	wsdiscovery "github.com/jfsmig/onvif/ws-discovery"
	"github.com/juju/errors"
)

var authInfo = networking.ClientAuth{
	Username: "admin",
	Password: "ollyhgqo",
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

		return runOneCamera(ctx, dev)
	}

	return errors.New("camera not found")
}

// runOneCamera starts a camera and its media sink and drives the camera
// through its controller, the way cams-agent does, only for a fixed duration
// and without retrying.
func runOneCamera(ctx context.Context, dev sdk.Appliance) error {
	camID := dev.GetUUID()
	busURL := "inproc://cams/cli/" + camID
	mediaURL := "inproc://cams/cli-media/" + camID

	// The sink is the passive side of the media bus and binds first; the camera
	// dials it once it has a stream to push.
	sink, err := NewTarMediaSink(mediaURL)
	if err != nil {
		return errors.Annotate(err, "media sink")
	}

	// The agent binds its command endpoint here, so the controller below can
	// dial it instead of racing the goroutine that serves it.
	agent, err := camagent.New(dev, busURL, mediaURL, camagent.NoRetry(),
		camagent.WithCredentials(authInfo.Username, authInfo.Password))
	if err != nil {
		if cerr := sink.Close(); cerr != nil {
			utils.Logger.Warn().Err(cerr).Msg("media sink close")
		}
		return errors.Annotate(err, "camera agent")
	}

	ctl, err := camctrl.New(camID, busURL)
	if err != nil {
		if cerr := agent.Close(); cerr != nil {
			utils.Logger.Warn().Err(cerr).Msg("camera agent close")
		}
		if cerr := sink.Close(); cerr != nil {
			utils.Logger.Warn().Err(cerr).Msg("media sink close")
		}
		return errors.Annotate(err, "camera controller")
	}
	defer func() {
		if cerr := ctl.Close(); cerr != nil {
			utils.Logger.Warn().Err(cerr).Msg("camera controller close")
		}
	}()

	utils.Logger.Info().Str("cam", camID).Str("bus", busURL).Msg("camera ready")

	// The sink is stopped after the agent, so the tail of the stream still has
	// somewhere to land.
	runCtx, stopAll := context.WithCancel(ctx)
	defer stopAll()

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		sink.Run(runCtx)
	}()
	go func() {
		defer wg.Done()
		agent.Run(runCtx)
	}()

	// Whatever happens below, both halves are released before returning.
	defer wg.Wait()
	defer stopAll()
	defer func() {
		if eerr := ctl.Exit(); eerr != nil {
			utils.Logger.Warn().Err(eerr).Str("action", "exit").Msg("stream command")
		}
	}()

	if err := ctl.Play(); err != nil {
		return errors.Annotate(err, "play")
	}
	utils.Logger.Info().Str("action", "start").Msg("stream command")

	select {
	case <-ctx.Done():
	case <-time.After(10 * time.Second):
	}
	utils.Logger.Info().Msg("stream ending")

	if err := ctl.Pause(); err != nil {
		return errors.Annotate(err, "pause")
	}
	utils.Logger.Info().Str("action", "stop").Msg("stream command")

	return nil
}

// tarMediaSink is the media upstream of the CLI: it pulls what the camera
// pushes and writes every frame into a tar archive, one entry per frame.
//
// Its state is touched only from the serving goroutine, so it needs no lock.
type tarMediaSink struct {
	puller *mediabus.Puller

	file    *os.File
	archive *tar.Writer

	packetCounter uint64
}

// NewTarMediaSink binds the media endpoint and opens the archive.
func NewTarMediaSink(bindURL string) (*tarMediaSink, error) {
	puller, err := mediabus.ListenPull("cli", bindURL)
	if err != nil {
		return nil, errors.Trace(err)
	}

	fout, err := ioutil.TempFile("", "cams-capture-*.tar")
	if err != nil {
		if cerr := puller.Close(); cerr != nil {
			utils.Logger.Warn().Err(cerr).Msg("media sink close")
		}
		return nil, errors.Annotate(err, "mktemp")
	}

	utils.Logger.Info().Str("path", fout.Name()).Msg("capture opened")

	return &tarMediaSink{
		puller:  puller,
		file:    fout,
		archive: tar.NewWriter(fout),
	}, nil
}

// Close releases the media endpoint of a sink that will never Run, and the
// archive with it.
func (sink *tarMediaSink) Close() error {
	err := sink.puller.Close()
	sink.closeArchive()
	return errors.Trace(err)
}

// Run consumes the media until the context is cancelled, then closes the
// archive so that it is readable.
func (sink *tarMediaSink) Run(ctx context.Context) {
	defer sink.closeArchive()
	sink.puller.Serve(ctx, sink.onFrame)
}

// onFrame names each frame by its type and its track, so that a capture of a
// multi-media session can be replayed one media at a time. The C++ replay
// harness matches on the extension alone and ignores the rest.
func (sink *tarMediaSink) onFrame(_ context.Context, f mediabus.Frame) error {
	var tag string
	switch f.Type {
	case mediabus.FrameSDP:
		tag = "sdp"
	case mediabus.FrameRTP:
		tag = "rtp"
	case mediabus.FrameRTCP:
		tag = "rtcp"
	default:
		return errors.NotValidf("media frame type %d", byte(f.Type))
	}
	return sink.writeFile(fmt.Sprintf("t%d.%s", f.Track, tag), f.Payload)
}

func (sink *tarMediaSink) closeArchive() {
	if sink.archive != nil {
		if err := sink.archive.Close(); err != nil {
			utils.Logger.Warn().Err(err).Msg("tar close")
		}
		sink.archive = nil
	}
	if sink.file != nil {
		if err := sink.file.Close(); err != nil {
			utils.Logger.Warn().Err(err).Msg("capture close")
		}
		sink.file = nil
	}
}

func (sink *tarMediaSink) writeFile(tag string, payload []byte) error {
	if sink.archive == nil {
		return errors.New("archive already closed")
	}

	sink.packetCounter++
	path := fmt.Sprintf("%06d", sink.packetCounter) + "." + tag
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
	if err := sink.archive.WriteHeader(&hdr); err != nil {
		return errors.Annotate(err, "tar header")
	}
	_, err := sink.archive.Write(payload)
	return errors.Annotate(err, "tar body")
}
