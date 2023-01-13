// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"github.com/jfsmig/cams/go/api/pb"
	"github.com/jfsmig/go-bags"
	"github.com/juju/errors"
	"sync"
	"time"
)

var (
	getSliceSize = uint32(100)
)

type streamRecord struct {
	StreamRegistration
	latUpdate time.Time
}

type registrarInMem struct {
	streams bags.SortedObj[string, *streamRecord]
	lock    sync.Mutex
}

func (sr streamRecord) PK() string { return sr.StreamID }

func NewRegistrarInMem() Registrar {
	return &registrarInMem{}
}

func (r *registrarInMem) Register(stream StreamRegistration) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if sr0, ok := r.streams.Get(stream.StreamID); !ok {
		// First discovery of the stream
		sr := streamRecord{StreamRegistration: stream, latUpdate: time.Now()}
		r.streams.Add(&sr)
		return nil
	} else if sr0.User != stream.User {
		return errors.New("device existing for another user")
	} else {
		sr0.latUpdate = time.Now()
		return nil
	}
}

func (r *registrarInMem) ListById(start string) ([]StreamRecord, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	out := make([]StreamRecord, 0, getSliceSize)
	for _, sr := range r.streams.Slice(start, getSliceSize) {
		out = append(out, StreamRecord{
			StreamID: sr.StreamID,
			User:     sr.User,
		})
	}
	return out, nil
}

func (hub *grpcHub) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.None, error) {
	err := hub.registrar.Register(StreamRegistration{req.Id.Stream, req.Id.User})
	if err != nil {
		return nil, err
	} else {
		return &pb.None{}, nil
	}
}
