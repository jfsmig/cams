// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"github.com/jfsmig/cams/proto"
	"github.com/juju/errors"
)

type localStreamEndpoint struct {
	streamID StreamID
	input    proto.StreamPlayer_MediaServer
	stop     chan bool
}

func NewStreamSource(src proto.StreamPlayer_MediaServer) StreamSource {
	return &localStreamEndpoint{
		input: src,
		stop:  make(chan bool, 1),
	}
}

func (ep *localStreamEndpoint) PK() string {
	return string(ep.streamID)
}

func (ep *localStreamEndpoint) Play() error {
	return errors.NotImplemented
}

func (ep *localStreamEndpoint) Stop() error {
	return errors.NotImplemented
}
