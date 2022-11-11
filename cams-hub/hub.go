// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"github.com/jfsmig/cams/api/pb"
	"github.com/jfsmig/cams/utils"
	"github.com/jfsmig/go-bags"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	pb.UnimplementedRegistrarServer
	pb.UnimplementedControllerServer

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

	utils.Logger.Info().Str("action", "start").Msg("hub")

	var cnx *grpc.Server
	var err error

	if len(config.PathCrt) <= 0 || len(config.PathKey) <= 0 {
		cnx, err = hub.config.ServeInsecure()
	} else {
		cnx, err = hub.config.ServeTLS()
	}
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", hub.config.ListenAddr)
	if err != nil {
		return err
	}

	pb.RegisterRegistrarServer(cnx, hub)
	pb.RegisterControllerServer(cnx, hub)

	hub.registrar = NewRegistrarInMem()

	return cnx.Serve(listener)
}

func (hub *grpcHub) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.None, error) {
	err := hub.registrar.Register(StreamRegistration{req.Id.Stream, req.Id.User})
	if err != nil {
		return nil, err
	} else {
		return &pb.None{}, nil
	}
}

func (hub *grpcHub) Control(stream pb.Controller_ControlServer) error {
	user, err := get[string](stream.Context(), "user")
	if err != nil {
		return status.Error(codes.InvalidArgument, "no agent id")
	}
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
			case CtrlCommandPlay: // Play a stream
			case CtrlCommandStop: // Stop a stream
			case CtrlCommandExit: // abort the
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

	//return status.Error(codes.Aborted, "An error occured")
	return nil
}

// An upload is starting.
// A banner is expected from the stream with the ID of the user and the ID of the stream
// Since the agent must wait for the PLAY command, there must be an expectation for that
func (hub *grpcHub) MediaUpload(stream pb.Controller_MediaUploadServer) error {
	// Extract the stream identifiers from the channel context
	var userId, streamId string
	var err error
	if userId, err = get[string](stream.Context(), "user"); err != nil {
		return err
	}
	if streamId, err = get[string](stream.Context(), "stream"); err != nil {
		return err
	}

	// Ensure the digital twin of the agent exists (it has been created at the provisionning step.
	// and that we create the ownly media upstream for that digital twin.
	agent, ok := hub.agent.Get(AgentID(userId))
	if !ok {
		return status.Error(codes.NotFound, "agent unknown")
	}

	upstream, err := agent.CreateStream(StreamID(streamId))
	if err != nil {
		return status.Error(codes.AlreadyExists, "stream found")
	}

	for running := true; running; {
		select {
		case req := <-upstream.requests:
			switch req {
			case MediaCommandExit:
				running = false
			default:
				utils.Logger.Warn().Msg("Unexpected command")
				running = false
			}
		default:
			if msg, err := stream.Recv(); err != nil {
				break
			} else {
				switch msg.Type {
				case pb.MediaFrameType_FrameType_RTP:
					// TODO(jfs): push the frame to its listeners
					utils.Logger.Info().Str("proto", "rtp").Msg("media")
				case pb.MediaFrameType_FrameType_RTCP:
					// TODO(jfs): push the frame to its listeners
					utils.Logger.Info().Str("proto", "rtcp").Msg("media")
				default:
					running = false
				}
			}
		}
	}

	// Close the subscribers
	agent.terminations <- upstream.PK()
	return err
}

func get[T any](ctx context.Context, k string) (T, error) {
	var zero T
	if v := ctx.Value(k); v == nil {
		return zero, status.Error(codes.InvalidArgument, "Missing field")
	} else if v, ok := v.(T); !ok {
		return zero, status.Error(codes.InvalidArgument, "Invalid field")
	} else {
		return v, nil
	}
}
