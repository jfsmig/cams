// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	pb2 "github.com/jfsmig/cams/go/api/pb"
	utils2 "github.com/jfsmig/cams/go/utils"
	"github.com/jfsmig/go-bags"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
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
	pb2.UnimplementedRegistrarServer
	pb2.UnimplementedControllerServer
	pb2.UnimplementedViewerServer

	config utils2.ServerConfig

	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup

	// Gathers the known streams
	registrar Registrar

	// Gather the established connections to agent on the field
	agent bags.SortedObj[AgentID, *AgentTwin]
}

func (hub *grpcHub) Register(ctx context.Context, req *pb2.RegisterRequest) (*pb2.None, error) {
	err := hub.registrar.Register(StreamRegistration{req.Id.Stream, req.Id.User})
	if err != nil {
		return nil, err
	} else {
		return &pb2.None{}, nil
	}
}

func (hub *grpcHub) Control(stream pb2.Controller_ControlServer) error {
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		err := status.Error(codes.AlreadyExists, "missing metadata")
		utils2.Logger.Warn().Str("action", "check").Err(err).Msg("hub control")
		return err
	}
	user := md.Get(utils2.KeyUser)[0]

	if hub.agent.Has(AgentID(user)) {
		err := status.Error(codes.AlreadyExists, "user agent already running")
		utils2.Logger.Warn().Str("action", "check").Err(err).Msg("hub control")
		return err
	}

	utils2.Logger.Trace().Str("action", "start").Msg("hub control")

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
			utils2.Logger.Info().Str("user", user).Str("stream", string(done)).Msg("terminated")
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

	return nil
}

// An upload is starting.
func (hub *grpcHub) MediaUpload(stream pb2.Controller_MediaUploadServer) error {
	// Extract the stream identifiers from the channel context
	var userId, streamId string
	var err error
	if userId, err = get[string](stream.Context(), utils2.KeyUser); err != nil {
		utils2.Logger.Warn().Str("action", "user").Err(err).Msg("hub media")
		return err
	}
	if streamId, err = get[string](stream.Context(), utils2.KeyStream); err != nil {
		utils2.Logger.Warn().Str("action", "stream").Str("user", userId).Err(err).Msg("hub media")
		return err
	}

	utils2.Logger.Trace().Str("action", "starting").Msg("hub media")

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
				utils2.Logger.Warn().Msg("Unexpected command")
				running = false
			}
		default:
			if msg, err := stream.Recv(); err != nil {
				break
			} else {
				switch msg.Type {
				case pb2.MediaFrameType_FrameType_RTP:
					// TODO(jfs): push the frame to its listeners
					utils2.Logger.Info().Str("proto", "rtp").Msg("media")
				case pb2.MediaFrameType_FrameType_RTCP:
					// TODO(jfs): push the frame to its listeners
					utils2.Logger.Info().Str("proto", "rtcp").Msg("media")
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

func (hub *grpcHub) Play(ctx context.Context, req *pb2.PlayRequest) (*pb2.None, error) {
	return nil, status.Error(codes.Unimplemented, "NYI")
}

func (hub *grpcHub) Pause(ctx context.Context, req *pb2.PauseRequest) (*pb2.None, error) {
	return nil, status.Error(codes.Unimplemented, "NYI")
}

func get[T any](ctx context.Context, k string) (T, error) {
	var zero T
	if v := ctx.Value(k); v == nil {
		return zero, status.Error(codes.InvalidArgument, "missing")
	} else if v, ok := v.(T); !ok {
		return zero, status.Error(codes.InvalidArgument, "zeroed")
	} else {
		return v, nil
	}
}
