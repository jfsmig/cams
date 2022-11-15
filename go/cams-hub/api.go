// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

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
