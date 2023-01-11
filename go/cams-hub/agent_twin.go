// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"github.com/jfsmig/cams/go/api/pb"
	"github.com/jfsmig/go-bags"
	"github.com/juju/errors"
	"strings"
	"sync"
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

const (
	MediaCommand_Exit = "exit"
)

type AgentTwin struct {
	agentID    AgentID
	downstream pb.Downstream_ControlServer

	// Control commands sent to the agents twin by the system
	requests chan CtrlCommand

	// notifications of terminated media goroutines
	terminations chan StreamID

	mediasLock      sync.Mutex
	mediasWaitGroup sync.WaitGroup
	medias          bags.SortedObj[StreamID, *agentStream]
}

type agentStream struct {
	streamID StreamID
	requests chan string
}

func NewAgentTwin(id AgentID, stream pb.Downstream_ControlServer) *AgentTwin {
	agent := AgentTwin{}
	agent.agentID = id
	agent.downstream = stream
	agent.requests = make(chan CtrlCommand, 1)
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

func NewAgentStream(id StreamID) *agentStream {
	return &agentStream{
		streamID: id,
		requests: make(chan string, 1),
	}
}

func (agent *AgentTwin) CreateStream(id StreamID) (*agentStream, error) {
	agent.mediasLock.Lock()
	defer agent.mediasLock.Unlock()
	src, ok := agent.medias.Get(id)
	if ok {
		return nil, errors.AlreadyExists
	}
	src = NewAgentStream(id)
	agent.medias.Add(src)
	return src, nil
}

func (as *agentStream) PK() StreamID {
	return as.streamID
}

func (as *agentStream) Exit() error {
	as.requests <- MediaCommand_Exit
	return nil
}
