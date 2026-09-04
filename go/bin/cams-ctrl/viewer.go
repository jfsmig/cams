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

	"github.com/jfsmig/cams/go/api/pb"
	"github.com/jfsmig/cams/go/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (hub *grpcHub) Play(ctx context.Context, req *pb.PlayRequest) (*pb.None, error) {
	utils.Logger.Info().Str("action", "play").Interface("cam", req).Msg("view")

	id, err := streamIdOf(req.GetId())
	if err != nil {
		return nil, err
	}
	return &pb.None{}, hub.viewerStreamAction(ctx, id.User, func(a *AgentTwin) error {
		return a.Play(ctx, id.Stream)
	})
}

func (hub *grpcHub) Pause(ctx context.Context, req *pb.PauseRequest) (*pb.None, error) {
	utils.Logger.Info().Str("action", "pause").Interface("cam", req).Msg("view")

	id, err := streamIdOf(req.GetId())
	if err != nil {
		return nil, err
	}
	return &pb.None{}, hub.viewerStreamAction(ctx, id.User, func(a *AgentTwin) error {
		return a.Stop(ctx, id.Stream)
	})
}

// streamIdOf validates the identity carried by a request.
//
// Id is a message field, so it is nil whenever a client omits it, and it used
// to be dereferenced straight away -- a two-field request was enough to panic
// the hub.
func streamIdOf(id *pb.StreamId) (*pb.StreamId, error) {
	if id == nil {
		return nil, status.Error(codes.InvalidArgument, "missing stream id")
	}
	if id.User == "" {
		return nil, status.Error(codes.InvalidArgument, "missing user")
	}
	if id.Stream == "" {
		return nil, status.Error(codes.InvalidArgument, "missing stream")
	}
	return id, nil
}

func (hub *grpcHub) viewerStreamAction(_ context.Context, agentId string, action func(*AgentTwin) error) error {
	agent := hub.getAgent(AgentID(agentId))
	if agent == nil {
		return status.Error(codes.NotFound, "agents not found")
	}

	return action(agent)
}
