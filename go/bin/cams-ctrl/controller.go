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
	"github.com/jfsmig/cams/go/api/pb"
	"github.com/jfsmig/cams/go/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (hub *grpcHub) Control(stream pb.Controller_ControlServer) error {
	user, err := userOf(stream.Context())
	if err != nil {
		utils.Logger.Warn().Str("action", "check").Err(err).Msg("hub ctrl")
		return err
	}

	utils.Logger.Trace().Str("user", user).Str("action", "start").Msg("hub ctrl")

	// One lookup-and-insert under one lock, rather than a Has followed by an
	// Add: two agents connecting as the same user at once both passed the test
	// and both inserted.
	agent := NewAgentTwin(AgentID(user))
	if !hub.addAgent(agent) {
		err := status.Error(codes.AlreadyExists, "user agents already running")
		utils.Logger.Warn().Str("user", user).Str("action", "check").Err(err).Msg("hub ctrl")
		return err
	}

	// wait for commands from outside, to propagate to the agents
	var sendErr error
	for running := true; running; {
		select {
		case <-stream.Context().Done():
			utils.Logger.Info().Str("user", user).Str("action", "shut").Msg("hub ctrl")
			running = false
		case cmd := <-agent.requests:
			switch cmd.cmdType {
			case CtrlCommandType_Play: // Play a stream
				sendErr = stream.Send(&pb.DownstreamControlRequest{Command: pb.DownstreamCommandType_DOWNSTREAM_COMMAND_TYPE_PLAY, StreamID: cmd.streamID})
			case CtrlCommandType_Stop: // Stop a stream
				sendErr = stream.Send(&pb.DownstreamControlRequest{Command: pb.DownstreamCommandType_DOWNSTREAM_COMMAND_TYPE_STOP, StreamID: cmd.streamID})
			case CtrlCommandType_Exit: // abort the connection to this agent
				running = false
			}
			// A stream that cannot be written to is finished; carrying on would
			// drop every later command in silence.
			if sendErr != nil {
				utils.Logger.Warn().Err(sendErr).Str("user", user).Msg("hub ctrl send")
				running = false
			}
		}
	}

	// Mark the twin as gone before unregistering it, so a viewer RPC already
	// holding the pointer is refused rather than left waiting.
	agent.close()
	hub.removeAgent(AgentID(user))

	// Returned bare. Wrapping it strips the gRPC status: juju's error is not a
	// GRPCStatus() type, so grpc rebuilds the status from the message and the
	// client sees codes.Unknown with an annotation chain for a message.
	return sendErr
}
