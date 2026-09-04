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

	"github.com/jfsmig/cams/go/utils"
	"github.com/juju/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// userOf reads the account an incoming call belongs to.
//
// It is the hub's only notion of identity, and it is an unverified string: an
// agent says who it is and the hub believes it. That is fine for a prototype
// and worth naming, because the half-built mTLS in utils.ServeTLS suggests
// otherwise.
func userOf(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.InvalidArgument, "missing metadata")
	}

	// Indexed without a length check, this was a panic any peer could reach by
	// omitting the header: the missing-metadata case above does not cover a
	// metadata block that simply lacks the key.
	users := md.Get(utils.KeyUser)
	if len(users) == 0 || users[0] == "" {
		return "", status.Error(codes.InvalidArgument, "missing user")
	}
	return users[0], nil
}

// statusOf turns a domain error into one a client can act on.
//
// The registry speaks in juju error types and the RPC boundary speaks in gRPC
// codes; this is the one place the two meet. Without it, a stream already owned
// by another account came back as codes.Unknown with a bare message, which no
// caller can distinguish from a bug.
func statusOf(err error) error {
	if err == nil {
		return nil
	}
	// Already a status: leave it alone.
	if _, ok := status.FromError(err); ok && status.Code(err) != codes.Unknown {
		return err
	}

	switch {
	case errors.Is(err, errors.AlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, errors.NotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, errors.NotValid):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, errors.NotSupported), errors.Is(err, errors.NotImplemented):
		return status.Error(codes.Unimplemented, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
