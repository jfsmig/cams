package main

import (
	pb2 "github.com/jfsmig/cams/go/api/pb"
	utils2 "github.com/jfsmig/cams/go/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func (hub *grpcHub) Control(stream pb2.Controller_ControlServer) error {
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		err := status.Error(codes.InvalidArgument, "missing metadata")
		utils2.Logger.Warn().Str("action", "check").Err(err).Msg("hub ctrl")
		return err
	}
	user := md.Get(utils2.KeyUser)[0]

	if hub.agents.Has(AgentID(user)) {
		err := status.Error(codes.AlreadyExists, "user agents already running")
		utils2.Logger.Warn().Str("user", user).Str("action", "check").Err(err).Msg("hub ctrl")
		return err
	}

	utils2.Logger.Trace().Str("user", user).Str("action", "start").Msg("hub ctrl")

	agent := NewAgentTwin(AgentID(user), stream)
	hub.agents.Add(agent)

	// wait for commands from outside, to propagate to the agents
	for running := true; running; {
		select {
		case <-stream.Context().Done():
			utils2.Logger.Info().Str("user", user).Str("action", "shut").Msg("hub ctrl")
			running = false
		case cmd := <-agent.requests:
			switch cmd.cmdType {
			case CtrlCommandType_Play: // Play a stream
				stream.Send(&pb2.DownstreamControlRequest{Command: pb2.DownstreamCommandType_DOWNSTREAM_COMMAND_TYPE_PLAY, StreamID: cmd.streamID})
			case CtrlCommandType_Stop: // Stop a stream
				stream.Send(&pb2.DownstreamControlRequest{Command: pb2.DownstreamCommandType_DOWNSTREAM_COMMAND_TYPE_STOP, StreamID: cmd.streamID})
			case CtrlCommandType_Exit: // abort the
				running = false
			}
		}
	}

	// Close the command channel
	close(agent.requests)

	// Unregister the AgentTwin
	hub.agents.Remove(AgentID(user))

	return nil
}
