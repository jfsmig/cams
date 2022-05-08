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

type AgentTwin struct {
	agentID    AgentID
	downstream pb.Controller_ControlServer

	// Control commands sent to the agents twin by the system
	requests chan CtrlCommand
}

func NewAgentTwin(id AgentID, stream pb.Controller_ControlServer) *AgentTwin {
	agent := AgentTwin{}
	agent.agentID = id
	agent.downstream = stream
	agent.requests = make(chan CtrlCommand, 1)
	return &agent
}

func (agent *AgentTwin) Play(streamID string) error {
	agent.requests <- CtrlCommand{CtrlCommandType_Play, streamID}
	return nil
}

func (agent *AgentTwin) Stop(streamID string) error {
	agent.requests <- CtrlCommand{CtrlCommandType_Stop, streamID}
	return nil
}

func (agent *AgentTwin) Exit() {
	agent.requests <- CtrlCommand{CtrlCommandType_Exit, ""}
}

func (agent *AgentTwin) PK() AgentID {
	return agent.agentID
}
