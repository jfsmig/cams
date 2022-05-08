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

	return &pb.None{}, hub.viewerStreamAction(req.Id.User, func(a *AgentTwin) error {
		return a.Play(req.Id.Stream)
	})
}

func (hub *grpcHub) Pause(ctx context.Context, req *pb.PauseRequest) (*pb.None, error) {
	utils.Logger.Info().Str("action", "pause").Interface("cam", req).Msg("view")

	return &pb.None{}, hub.viewerStreamAction(req.Id.User, func(a *AgentTwin) error {
		return a.Stop(req.Id.Stream)
	})
}

func (hub *grpcHub) viewerStreamAction(agentId string, action func(*AgentTwin) error) error {
	if !hub.agents.Has(AgentID(agentId)) {
		return status.Error(codes.NotFound, "agents not found")
	}
	agent, _ := hub.agents.Get(AgentID(agentId))

	if err := action(agent); err != nil {
		return err
	} else {
		return nil
	}
}
