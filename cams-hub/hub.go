// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"github.com/jfsmig/cams/proto"
	"github.com/jfsmig/cams/utils"
	"github.com/jfsmig/go-bags"
	"github.com/juju/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"net"
	"strings"
	"sync"
)

type AgentID string
type StreamID string

type TLSConfig struct {
	PathCrt string `json:"crt"`
	PathKey string `json:"key"`
}

type grpcHub struct {
	proto.UnimplementedRegistrarServer
	proto.UnimplementedControllerServer
	proto.UnimplementedConsumerServer

	config utils.ServerConfig

	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup

	// Gathers the known streams
	registrar Registrar

	// Gather the established connections to agent on the field
	agent bags.SortedObj[AgentID, *AgentTwin]
}

func runHub(ctx context.Context, config utils.ServerConfig) error {
	hub := &grpcHub{
		config: config,
	}

	cnx, err := hub.config.ServeTLS()
	if err != nil {
		return err
	}

	listener, err := net.Listen("", hub.config.ListenAddr)
	if err != nil {
		return err
	}

	proto.RegisterRegistrarServer(cnx, hub)
	proto.RegisterControllerServer(cnx, hub)
	proto.RegisterConsumerServer(cnx, hub)

	hub.registrar = NewRegistrarInMem()

	return cnx.Serve(listener)
}

func (hub *grpcHub) Register(ctx context.Context, req *proto.RegisterRequest) (*proto.RegisterReply, error) {
	err := hub.registrar.Register(StreamRegistration{req.Id.Stream, req.Id.Stream})
	return &proto.RegisterReply{
		Status: &proto.Status{Code: 202, Status: "registered"},
	}, err
}

func (hub *grpcHub) Control(stream proto.Controller_ControlServer) error {
	// Consume the banner
	banner, err := stream.Recv()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return errors.Annotate(err, "stream read")
	}

	// locate the agent is any
	if banner.GetBanner() == nil {
		return status.Error(codes.InvalidArgument, "expected banner")
	}
	user := banner.GetBanner().GetUser()
	if hub.agent.Has(AgentID(user)) {
		return status.Error(codes.AlreadyExists, "agent known")
	}

	agent := NewAgentTwin(AgentID(user), stream)

	// wait for commands from outside, to propagate to the agent
	for running := true; running; {
		select {
		case cmd := <-agent.requests:
			tokens := strings.Split(cmd, " ")
			switch tokens[0] {
			case CommandPlay: // Play a stream
			case CommandStop: // Stop a stream
			case CommandExit: // abort the
				running = false
			}
		case done := <-agent.terminations:
			agent.medias.Remove(done)
			utils.Logger.Info().Str("user", user).Str("stream", string(done)).Msg("terminated")
		}
	}

	// Close all the media streams
	for _, media := range agent.medias {
		media.Exit()
	}
	agent.mediasWaitGroup.Wait()

	// Close the command channel
	close(agent.requests)

	// Unregister the AgentTwin
	hub.agent.Remove(AgentID(user))

	if err == nil {
		return nil
	}
	return status.Error(codes.Aborted, "An error occured")
}

// An upload is starting.
// A banner is expected from the stream with the ID of the user and the ID of the stream
// Since the agent must wait for the PLAY command, there must be an expectation for that
func (hub *grpcHub) MediaUpload(stream proto.Controller_MediaUploadServer) error {
	msg, err := stream.Recv()
	if err != nil {
		return err
	}
	if msg.GetBanner() == nil {
		return status.Error(codes.InvalidArgument, "expected banner")
	}

	agent, ok := hub.agent.Get(AgentID(msg.GetBanner().GetUser()))
	if !ok {
		return status.Error(codes.NotFound, "no such agent")
	}

	src, err := agent.Create(StreamID(msg.GetBanner().GetStream()))

	for running := true; running; {
		select {
		case req := <-src.requests:
			switch req {
			case CommandExit:
				running = false
			default:
				utils.Logger.Warn().Msg("Unexpected command")
				running = false
			}
		default:
			if msg, err = stream.Recv(); err != nil {
				break
			}
			// TODO(jfs): push the frame to its listeners
		}
	}

	// Close the subscribers
	agent.terminations <- src.PK()
	return err
}

func (hub *grpcHub) Play(id *proto.StreamId, req proto.Consumer_PlayServer) error {
	agent, ok := hub.agent.Get(AgentID(id.GetUser()))
	if !ok {
		return status.Error(codes.NotFound, "no such agent")
	}

	_, ok = agent.medias.Get(StreamID(id.Stream))
	if !ok {
		return status.Error(codes.NotFound, "no such stream")
	}

	return status.Error(codes.Unimplemented, "NYI")
}
