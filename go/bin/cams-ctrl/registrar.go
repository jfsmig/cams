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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// registrationTTL is how long a stream stays known without being announced
	// again. The agents re-register every RegisterPeriod, so anything much
	// older than that has gone.
	registrationTTL = 5 * time.Minute

	// defaultMaxStreams caps what one hub will remember. Register is reachable
	// by any peer that can open a connection, and every call used to add an
	// entry that nothing ever removed, so the set was an unbounded allocation
	// controlled from outside.
	defaultMaxStreams = 10000
)

type streamRecord struct {
	StreamRegistration
	latUpdate time.Time
}

type registrarInMem struct {
	streams bags.SortedObj[string, *streamRecord]
	lock    sync.Mutex

	// maxStreams bounds the set. A field rather than a constant so that a test
	// can exercise the limit without paying for it: the set is a sorted slice
	// that re-sorts on insert, so filling it to the production cap costs
	// quadratic time.
	maxStreams int
}

func (sr streamRecord) PK() string { return sr.StreamID }

func NewRegistrarInMem() Registrar {
	return &registrarInMem{maxStreams: defaultMaxStreams}
}

func (r *registrarInMem) Register(stream StreamRegistration) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if sr0, ok := r.streams.Get(stream.StreamID); ok {
		if sr0.User != stream.User {
			return errors.AlreadyExistsf("stream %q, for another user", stream.StreamID)
		}
		sr0.latUpdate = time.Now()
		return nil
	}

	// A first sighting. Make room before making the entry: latUpdate exists to
	// support exactly this and had no reader until now.
	r.expireLocked(time.Now())

	if len(r.streams) >= r.maxStreams {
		return errors.Errorf("the registry is full at %d streams", r.maxStreams)
	}

	sr := streamRecord{StreamRegistration: stream, latUpdate: time.Now()}
	r.streams.Add(&sr)
	return nil
}

// expireLocked forgets the streams nobody has announced for a while. The caller
// holds the lock.
func (r *registrarInMem) expireLocked(now time.Time) {
	// Backwards, because Remove compacts the slice: indices above the removed
	// one shift down by exactly one, which leaves the not-yet-visited ones
	// valid.
	for i := len(r.streams); i > 0; i-- {
		sr := r.streams[i-1]
		if now.Sub(sr.latUpdate) > registrationTTL {
			r.streams.Remove(sr.PK())
		}
	}
}

func (hub *grpcHub) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.None, error) {
	id, err := streamIdOf(req.GetId())
	if err != nil {
		return nil, err
	}

	// The account comes from the metadata, the way Control takes it, and the
	// body has to agree. Trusting the body alone let any peer register streams
	// under any account it named.
	user, err := userOf(ctx)
	if err != nil {
		return nil, err
	}
	if id.User != user {
		return nil, status.Errorf(codes.PermissionDenied,
			"registration for %q from %q", id.User, user)
	}

	// Keyed, because StreamRegistration and pb.StreamId order their two string
	// fields the other way round: an unkeyed literal here was one innocuous
	// field reorder away from silently swapping user and stream.
	reg := StreamRegistration{StreamID: id.Stream, User: id.User}
	if err := hub.registrar.Register(reg); err != nil {
		return nil, statusOf(err)
	}
	return &pb.None{}, nil
}
