// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import "context"

type Registrar interface {
	Register(stream StreamRegistration) error

	ListById(start string) ([]StreamRecord, error)
}

type StreamRecord struct {
	StreamID string
	User     string
}

type StreamRegistration struct {
	StreamID string
	User     string
}

type StreamPlayer interface {
	Run(ctx context.Context) error
	Register(src StreamSource) error
	Unregister(src StreamSource) error
}

type StreamSource interface {
	PK() string
	Run() error

	Play() error
	Stop() error
}
