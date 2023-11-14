// Copyright (c) 2022-2024 The authors (see the AUTHORS file)
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"context"
	"sync"
	"time"

	"github.com/jfsmig/cams/go/api/pb"
	"github.com/jfsmig/go-bags"
	"github.com/juju/errors"
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
