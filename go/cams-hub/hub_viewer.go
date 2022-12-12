package main

import (
	"context"
	"github.com/jfsmig/cams/go/api/pb"
	"github.com/jfsmig/cams/go/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (hub *grpcHub) Play(ctx context.Context, req *pb.PlayRequest) (*pb.None, error) {
	utils.Logger.Info().Str("action", "play").Msg("view")

	return &pb.None{}, hub.viewerAction(req.Id.User, req.Id.Stream, func(a *AgentTwin, s string) error {
		return a.Play(s)
	})
}

func (hub *grpcHub) Pause(ctx context.Context, req *pb.PauseRequest) (*pb.None, error) {
	utils.Logger.Info().Str("action", "pause").Msg("view")

	return &pb.None{}, hub.viewerAction(req.Id.User, req.Id.Stream, func(a *AgentTwin, s string) error {
		return a.Stop(s)
	})
}

func (hub *grpcHub) viewerAction(agentId, streamId string, action func(*AgentTwin, string) error) error {
	if !hub.agent.Has(AgentID(agentId)) {
		return status.Error(codes.NotFound, "agent not found")
	}
	agent, _ := hub.agent.Get(AgentID(agentId))
	if !agent.medias.Has(StreamID(streamId)) {
		return status.Error(codes.NotFound, "stream not found")
	}

	if err := action(agent, streamId); err != nil {
		return err
	} else {
		return nil
	}
}
