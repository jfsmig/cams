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
	"context"
	"os"
	"os/signal"

	"github.com/jfsmig/cams/go/camagent"
	"github.com/jfsmig/cams/go/camctrl"
	"github.com/jfsmig/cams/go/lanagent"
	"github.com/jfsmig/cams/go/lanctrl"
	"github.com/jfsmig/cams/go/upagent"
	"github.com/jfsmig/cams/go/utils"
	"github.com/jfsmig/onvif/v2/sdk"
	"github.com/juju/errors"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

func main() {
	var flagSpeed bool

	cmd := &cobra.Command{
		Use:   "cams-agent",
		Short: "Cams Agent",
		Long:  "LAN agent for OnVif cameras",
		//Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			zerolog.SetGlobalLevel(zerolog.TraceLevel)
			// FIXME(jfs): load an external configuration file or CLI options
			cfg := DefaultConfig()
			if flagSpeed {
				cfg.RegisterPeriod = 1
				cfg.ScanPeriod = 0
				cfg.CheckPeriod = 1
			}
			ctx, cancel := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)
			defer cancel()

			return runAgent(ctx, cfg)
		},
	}

	cmd.Flags().BoolVarP(&flagSpeed, "speed", "s", false, "TEST with fast loops")

	if err := cmd.Execute(); err != nil {
		utils.Logger.Fatal().Err(err).Str("action", "aborting").Msg("agent")
	} else {
		utils.Logger.Info().Str("action", "Exiting").Msg("agent")
	}
}

// camBusURL and camMediaURL are the bus endpoints of one camera: the first
// carries its commands, the second its media. The inproc registry is
// process-global and matched on exact equality, so the names are kept under a
// prefix of ours.
func camBusURL(camID string) string   { return "inproc://cams/control/" + camID }
func camMediaURL(camID string) string { return "inproc://cams/media/" + camID }

// cameraBundle is one camera as the LAN agent sees it: the agent that captures
// the media and the upstream that forwards it, started and stopped together.
//
// The two halves are paired here because the LAN agent owns cameras, not media
// upstreams, and neither half is useful without the other.
type cameraBundle struct {
	agent *camagent.Agent
	sink  *grpcMediaSink
}

// Run serves both halves and returns when either is done. An ensemble rather
// than a swarm: a camera whose upstream has stopped has nowhere to push, and an
// upstream whose camera has exited has nothing to forward.
func (b *cameraBundle) Run(ctx context.Context) {
	utils.EnsembleRun(ctx, b.sink.Run, b.agent.Run)
}

// Close releases both halves. It is what the LAN agent calls for a camera it
// built and then decided not to start.
func (b *cameraBundle) Close() error {
	// Both are attempted whatever the first one says, since both hold an
	// endpoint that would otherwise leak.
	agentErr := b.agent.Close()
	sinkErr := b.sink.Close()
	if agentErr != nil {
		return errors.Annotate(agentErr, "camera agent")
	}
	return errors.Annotate(sinkErr, "media sink")
}

// newCameraFactory is the only place in the process that knows how a camera is
// assembled. Handing this closure to the LAN agent is what keeps camagent out
// of every other package.
func newCameraFactory(cfg AgentConfig) lanagent.CameraFactory {
	return func(appliance sdk.Appliance) (camctrl.CameraController, lanagent.CameraAgent, error) {
		camID := appliance.GetUUID()
		busURL := camBusURL(camID)
		mediaURL := camMediaURL(camID)

		// The media upstream binds first: it is the passive side, and the
		// camera dials it once it has a stream to push.
		sink, err := NewGrpcMediaSink(cfg.User, camID, cfg.UpstreamMedia.Address, mediaURL)
		if err != nil {
			return nil, nil, errors.Annotate(err, "media sink")
		}

		// The agent binds too, so the controller below can dial right away.
		agent, err := camagent.New(camagent.FromSDK(appliance), busURL, mediaURL,
			camagent.WithCredentials(cfg.CameraUser, cfg.CameraPassword))
		if err != nil {
			if cerr := sink.Close(); cerr != nil {
				utils.Logger.Warn().Err(cerr).Str("cam", camID).Msg("media sink close")
			}
			return nil, nil, errors.Annotate(err, "camera agent")
		}

		bundle := &cameraBundle{agent: agent, sink: sink}

		ctl, err := camctrl.New(camID, busURL)
		if err != nil {
			if cerr := bundle.Close(); cerr != nil {
				utils.Logger.Warn().Err(cerr).Str("cam", camID).Msg("camera close")
			}
			return nil, nil, errors.Annotate(err, "camera controller")
		}

		return ctl, bundle, nil
	}
}

// lanBusURL is the bus endpoint of the LAN agent. The inproc registry is
// process-global and matched on exact equality, so the names are kept under a
// prefix of ours.
const (
	lanBusURL = "inproc://cams/lan"
	upBusURL  = "inproc://cams/upstream"
)

func runAgent(ctx context.Context, cfg AgentConfig) error {
	// The LAN agent binds its endpoint here, so the controller below can dial
	// it before anything is scheduled.
	lan, err := lanagent.New(cfg.lan(), newCameraFactory(cfg), lanBusURL)
	if err != nil {
		return errors.Annotate(err, "lan agent")
	}

	lanCli, err := lanctrl.New(lanBusURL)
	if err != nil {
		if cerr := lan.Close(); cerr != nil {
			utils.Logger.Warn().Err(cerr).Msg("lan agent close")
		}
		return errors.Annotate(err, "lan controller")
	}
	defer func() {
		if cerr := lanCli.Close(); cerr != nil {
			utils.Logger.Warn().Err(cerr).Msg("lan controller close")
		}
	}()

	// The upstream agent reaches the LAN through its controller and knows
	// nothing else about it.
	upstream, err := upagent.New(cfg.upstream(), lanCli, upBusURL)
	if err != nil {
		return errors.Annotate(err, "upstream agent")
	}

	utils.Logger.Info().Str("action", "starting").Msg("agent")

	utils.EnsembleRun(ctx, upstream.Run, lan.Run)

	return nil
}
