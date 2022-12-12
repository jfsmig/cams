package main

import (
	"github.com/jfsmig/cams/go/api/pb"
	"github.com/jfsmig/cams/go/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"strings"
)

func (hub *grpcHub) Control(stream pb.Downstream_ControlServer) error {
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		err := status.Error(codes.InvalidArgument, "missing metadata")
		utils.Logger.Warn().Str("action", "check").Err(err).Msg("hub control")
		return err
	}
	user := md.Get(utils.KeyUser)[0]

	if hub.agent.Has(AgentID(user)) {
		err := status.Error(codes.AlreadyExists, "user agent already running")
		utils.Logger.Warn().Str("action", "check").Err(err).Msg("hub control")
		return err
	}

	utils.Logger.Trace().Str("action", "start").Msg("hub control")

	agent := NewAgentTwin(AgentID(user), stream)

	// wait for commands from outside, to propagate to the agent
	for running := true; running; {
		select {
		case cmd := <-agent.requests:
			tokens := strings.Split(cmd, " ")
			switch tokens[0] {
			case CtrlCommandPlay: // Play a stream
				stream.Send(&pb.DownstreamControlRequest{Command: pb.DownstreamCommandType_DOWNSTREAM_COMMAND_TYPE_PLAY, StreamID: tokens[1]})
			case CtrlCommandStop: // Stop a stream
				stream.Send(&pb.DownstreamControlRequest{Command: pb.DownstreamCommandType_DOWNSTREAM_COMMAND_TYPE_STOP, StreamID: tokens[1]})
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

	return nil
}

// An upload is starting.
func (hub *grpcHub) Media(stream pb.Downstream_MediaUploadServer) error {
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		err := status.Error(codes.InvalidArgument, "missing metadata")
		utils.Logger.Warn().Str("action", "check").Err(err).Msg("hub media")
		return err
	}
	userId := md.Get(utils.KeyUser)[0]
	streamId := md.Get(utils.KeyStream)[0]

	utils.Logger.Trace().Str("action", "starting").
		Str("user", userId).Str("cam", streamId).
		Msg("hub media")

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
				case pb.DownstreamMediaFrameType_DOWNSTREAM_MEDIA_FRAME_TYPE_RTP:
					// TODO(jfs): push the frame to its listeners
					utils.Logger.Info().Str("proto", "rtp").Msg("hub media")
				case pb.DownstreamMediaFrameType_DOWNSTREAM_MEDIA_FRAME_TYPE_RTCP:
					// TODO(jfs): push the frame to its listeners
					utils.Logger.Info().Str("proto", "rtcp").Msg("hub media")
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
