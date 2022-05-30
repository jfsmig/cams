// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"github.com/jfsmig/cams/proto"
	"github.com/juju/errors"
)

type StreamPlayer interface {
	PK() string
	Stop() error
	Run() error
}

func NewStreamPlayer(proto.Controller_MediaServer) StreamPlayer {
	return &localStreamEndpoint{}
}

type localStreamEndpoint struct {
	streamID StreamID
	input    proto.Controller_MediaServer
	stop     chan bool
}

func (ep *localStreamEndpoint) PK() string {
	return string(ep.streamID)
}

func (ep *localStreamEndpoint) Stop() error {
	return errors.NotImplemented
}

func (ep *localStreamEndpoint) Run() error {
	return errors.NotImplemented
}
