// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

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
