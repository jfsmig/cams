// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

type registrarInMem struct {
	streams map[string]Stream
}

func NewRegistrarInMem() Registrar {
	return &registrarInMem{
		streams: make(map[string]Stream),
	}
}

func (r *registrarInMem) Register(stream Stream) {
	//TODO implement me
	panic("implement me")
}

func (r *registrarInMem) ListById(start string) {
	//TODO implement me
	panic("implement me")
}
