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
	"net"
	"os"
	"os/signal"
	"sync"

	"github.com/jfsmig/cams/go/api/pb"
	"github.com/jfsmig/cams/go/utils"
	"github.com/jfsmig/go-bags"
	"github.com/juju/errors"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

type Registrar interface {
	Register(stream StreamRegistration) error
}

type StreamRegistration struct {
	StreamID string
	User     string
}

type AgentID string
type StreamID string

type TLSConfig struct {
	PathCrt string `json:"crt"`
	PathKey string `json:"key"`
}

type grpcHub struct {
	pb.UnimplementedRegistrarServer
	pb.UnimplementedControllerServer
	pb.UnimplementedViewerServer
	pb.UnimplementedUploaderServer

	config utils.ServerConfig

	// Gathers the known streams. Its own implementation is locked; this field
	// is set once, at construction.
	registrar Registrar

	// Gather the established connections to agents on the field.
	//
	// One goroutine per connected agent adds and removes, and every viewer RPC
	// reads, so the slice underneath needs a lock of its own -- reach it only
	// through the accessors below.
	agentsLock sync.Mutex
	agents     bags.SortedObj[AgentID, *AgentTwin]
}

// addAgent registers a freshly connected agent, and reports false when one is
// already connected for that identity.
func (hub *grpcHub) addAgent(agent *AgentTwin) bool {
	hub.agentsLock.Lock()
	defer hub.agentsLock.Unlock()

	if hub.agents.Has(agent.PK()) {
		return false
	}
	hub.agents.Add(agent)
	return true
}

// removeAgent forgets a departing agent.
func (hub *grpcHub) removeAgent(id AgentID) {
	hub.agentsLock.Lock()
	defer hub.agentsLock.Unlock()
	hub.agents.Remove(id)
}

// getAgent returns the agent connected for an identity, or nil.
//
// One lookup, not a Has followed by a Get: between the two the agent may have
// disconnected, and the old pair returned a nil twin that the caller then
// dereferenced.
func (hub *grpcHub) getAgent(id AgentID) *AgentTwin {
	hub.agentsLock.Lock()
	defer hub.agentsLock.Unlock()

	agent, ok := hub.agents.Get(id)
	if !ok {
		return nil
	}
	return agent
}

func main() {
	cmd := &cobra.Command{
		Use:   "cams-ctrl",
		Short: "Cams Hub",
		Long:  "Hub / Upstream for Cams Agent",
		//Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// FIXME(jfs): load an external configuration file or CLI options
			cfg := utils.ServerConfig{
				ListenAddr: "127.0.0.1:6000",
				PathCrt:    "",
				PathKey:    "",
			}

			ctx, cancel := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)
			defer cancel()

			return runHub(ctx, cfg)
		},
	}

	if err := cmd.Execute(); err != nil {
		utils.Logger.Fatal().Err(err).Msg("Aborting")
	} else {
		utils.Logger.Info().Msg("Exiting")
	}
}

func runHub(ctx context.Context, config utils.ServerConfig) error {
	hub := &grpcHub{
		config: config,
		// Built here rather than inside a serving goroutine: it used to be
		// assigned from two of them, which raced, threw one registry away, and
		// left a window where Register could be served before either ran.
		registrar: NewRegistrarInMem(),
	}

	utils.Logger.Info().Str("action", "start").Msg("hub")

	var server *grpc.Server
	var err error
	if len(config.PathCrt) <= 0 || len(config.PathKey) <= 0 {
		server, err = hub.config.ServeInsecure()
	} else {
		server, err = hub.config.ServeTLS()
	}
	if err != nil {
		return errors.Annotate(err, "grpc server")
	}

	// One server and one port for all four services.
	//
	// There used to be two of each, both bound to the same ListenAddr, so the
	// second Listen failed with EADDRINUSE and the hub could not start at all.
	//
	// Uploader is registered here but refuses every call: the media plane is a
	// separate process, cams-rtp2hls, on its own port -- which is why the
	// agent's UpstreamMedia.Address defaults to cams-agent's DefaultMediaAddr
	// and not to this one. Registering it anyway is what turns a
	// misconfiguration into an Unimplemented rather than a connection to a
	// service that is not there.
	pb.RegisterRegistrarServer(server, hub)
	pb.RegisterControllerServer(server, hub)
	pb.RegisterViewerServer(server, hub)
	pb.RegisterUploaderServer(server, hub)

	listener, err := net.Listen("tcp", hub.config.ListenAddr)
	if err != nil {
		return errors.Annotatef(err, "listen %s", hub.config.ListenAddr)
	}

	// An ensemble, not a swarm: the two members only make sense together, and
	// either one finishing means the hub is done.
	//
	// With a swarm this hung. A swarm waits for every member, so if Serve
	// returned on its own -- an Accept error that grpc does not consider
	// temporary -- the watchdog stayed parked on its context and nothing was
	// left to cancel it. The hub was dead, the port was closed, and the process
	// sat there until it was signalled.
	utils.EnsembleRun(ctx,
		func(c context.Context) {
			<-c.Done()
			utils.Logger.Info().Str("action", "kill").Msg("hub")
			// GracefulStop closes the listener, so it is not closed here as
			// well.
			server.GracefulStop()
		},
		func(c context.Context) {
			if serr := server.Serve(listener); serr != nil && !errors.Is(serr, grpc.ErrServerStopped) {
				utils.Logger.Warn().Err(serr).Msg("hub serve error")
			}
		},
	)

	return nil
}
