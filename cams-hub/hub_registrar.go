package main

import (
	"context"
	"github.com/jfsmig/cams/api/pb"
)

func (hub *grpcHub) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.None, error) {
	err := hub.registrar.Register(StreamRegistration{req.Id.Stream, req.Id.User})
	if err != nil {
		return nil, err
	} else {
		return &pb.None{}, nil
	}
}
