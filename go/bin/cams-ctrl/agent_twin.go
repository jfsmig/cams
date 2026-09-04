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

	"github.com/juju/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CtrlCommandType uint32

type CtrlCommand struct {
	cmdType  CtrlCommandType
	streamID string
}

const (
	CtrlCommandType_Play CtrlCommandType = iota
	CtrlCommandType_Stop
	CtrlCommandType_Exit
)

// ErrAgentGone reports a command aimed at an agent whose connection has ended.
var ErrAgentGone = errors.New("the agent has disconnected")

type AgentTwin struct {
	agentID AgentID

	// Control commands sent to the agents twin by the system
	requests chan CtrlCommand

	// done is closed by the goroutine serving this agent's stream when it
	// returns. It is what lets a command refuse instead of blocking on a queue
	// nobody reads any more.
	done chan struct{}
}

func NewAgentTwin(id AgentID) *AgentTwin {
	return &AgentTwin{
		agentID:  id,
		requests: make(chan CtrlCommand, 1),
		done:     make(chan struct{}),
	}
}

// close marks the twin as gone. Only the goroutine serving the stream calls it,
// exactly once.
//
// The request channel is deliberately left open: it used to be closed here, and
// a viewer RPC that was mid-send at that moment panicked on a send to a closed
// channel. Nothing else holds the channel, so it goes with the twin.
func (agent *AgentTwin) close() { close(agent.done) }

// send queues one command for the agent.
//
// It gives up when the caller's context is done, so a viewer RPC cannot be held
// by a slow agent, and when the agent has gone, so it cannot be held forever by
// a queue with no reader.
func (agent *AgentTwin) send(ctx context.Context, cmd CtrlCommand) error {
	// The departure is tested first, on its own. A closed channel is
	// permanently ready, so inside the select below it competes with a send
	// that has room -- and a select picks uniformly among ready cases, so a
	// departed agent would accept about half the commands aimed at it and
	// report success.
	select {
	case <-agent.done:
		return status.Error(codes.Unavailable, ErrAgentGone.Error())
	default:
	}

	select {
	case agent.requests <- cmd:
		return nil
	case <-agent.done:
		return status.Error(codes.Unavailable, ErrAgentGone.Error())
	case <-ctx.Done():
		return status.FromContextError(ctx.Err()).Err()
	}
}

func (agent *AgentTwin) Play(ctx context.Context, streamID string) error {
	return agent.send(ctx, CtrlCommand{CtrlCommandType_Play, streamID})
}

func (agent *AgentTwin) Stop(ctx context.Context, streamID string) error {
	return agent.send(ctx, CtrlCommand{CtrlCommandType_Stop, streamID})
}

func (agent *AgentTwin) PK() AgentID {
	return agent.agentID
}
