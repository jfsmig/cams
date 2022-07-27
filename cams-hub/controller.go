// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"github.com/jfsmig/cams/proto"
	"github.com/jfsmig/go-bags"
	"github.com/juju/errors"
	"strings"
	"sync"
)

const (
	CommandPlay = "play"
	CommandStop = "stop"
	CommandExit = "exit"
)

type AgentTwin struct {
	agentID    AgentID
	downstream proto.Controller_ControlServer

	// Control commands sent to the agent twin by the system
	requests chan string

	// notifications of terminated media goroutines
	terminations chan StreamID

	mediasLock      sync.Mutex
	mediasWaitGroup sync.WaitGroup
	medias          bags.SortedObj[StreamID, *agentStream]
}

type agentStream struct {
	streamID   StreamID
	downstream proto.Controller_MediaUploadServer
}

func NewAgentTwin(id AgentID, stream proto.Controller_ControlServer) *AgentTwin {
	agent := AgentTwin{}
	agent.agentID = id
	agent.downstream = stream
	agent.requests = make(chan string, 1)
	agent.terminations = make(chan StreamID, 32)
	return &agent
}

func _command(action string, agentToken string, args ...string) string {
	sb := strings.Builder{}
	sb.WriteString(action)
	sb.WriteRune(' ')
	sb.WriteString(agentToken)
	for _, arg := range args {
		sb.WriteRune(' ')
		sb.WriteString(arg)
	}
	return sb.String()
}

func (agent *AgentTwin) Play(streamID string) error {
	agent.requests <- _command(CommandPlay, string(agent.PK()), streamID)
	return nil
}

func (agent *AgentTwin) Stop(streamID string) error {
	agent.requests <- _command(CommandStop, string(agent.PK()), streamID)
	return nil
}

func (agent *AgentTwin) Exit() {
	agent.requests <- _command(CommandExit, string(agent.PK()))
}

func (agent *AgentTwin) PK() AgentID {
	return agent.agentID
}

func (as *agentStream) PK() StreamID {
	return as.streamID
}

func (as *agentStream) Exit() error {
	return errors.NotImplemented
}
