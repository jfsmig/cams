// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"github.com/jfsmig/cams/proto"
	"github.com/juju/errors"
	"github.com/qmuntal/stateless"
	"io"
	"strings"
)

const (
	stateInit    = "init"
	stateIdle    = "idle"
	stateReady   = "ready"
	stateCommand = "cmd"
)

const (
	triggerStart  = "start"
	triggerBanner = "banner"
	triggerReply  = "reply"
)

type AgentController interface {
	// PK makes the object sortable
	PK() string

	// Play Asks the remote agent to start playing a media stream
	Play(streamID string) error

	// Stop asks the remote agent to stop playing a media stream.
	//      The stream itself SHOULD NOT be immediately closed on
	//      the server side
	Stop(streamID string) error

	// Run executes the goroutine that makes the remote agent alive
	//     on the server
	Run() error
}

type agentTwin struct {
	streamID   StreamID
	fsm        *stateless.StateMachine
	requests   chan string
	downstream proto.Controller_ControlServer
}

func NewAgenController(stream proto.Controller_ControlServer) AgentController {
	agent := agentTwin{}
	agent.fsm = stateless.NewStateMachine(stateInit)
	agent.downstream = stream
	agent.requests = make(chan string, 32)

	agent.fsm.Configure(stateInit).
		Permit(triggerStart, stateIdle)

	agent.fsm.Configure(stateIdle).
		OnActive(func(_ context.Context) error {
			// TODO(jfs): consume the banner
			_, err := agent.downstream.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				return errors.Annotate(err, "stream read")
			} else {

			}
			return nil
		}).
		Permit(triggerBanner, stateReady)

	agent.fsm.Configure(stateReady).
		OnActive(func(_ context.Context) error {
			// TODO(jfs): consume the request channels
			return nil
		}).
		Permit(triggerBanner, stateReady)

	agent.fsm.Configure(stateCommand).
		Permit(triggerReply, stateReady)

	return &agent
}

func _command(action string, streamID string) string {
	sb := strings.Builder{}
	sb.WriteString(action)
	sb.WriteRune(' ')
	sb.WriteString(streamID)
	return sb.String()
}

func (agent *agentTwin) Play(streamID string) error {
	agent.requests <- _command("play", streamID)
	return nil
}

func (agent *agentTwin) Stop(streamID string) error {
	agent.requests <- _command("stop", streamID)
	return nil
}

func (agent *agentTwin) Run() error {
	agent.fsm.Fire(triggerStart)
	return nil
}

func (agent *agentTwin) PK() string {
	return string(agent.streamID)
}
