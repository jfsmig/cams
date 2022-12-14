// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"sync"

	"github.com/jfsmig/cams/go/api/pb"
	"github.com/jfsmig/cams/go/utils"
	"github.com/jfsmig/go-bags"
)

type AgentID string
type StreamID string

type TLSConfig struct {
	PathCrt string `json:"crt"`
	PathKey string `json:"key"`
}

type grpcHub struct {
	pb.UnimplementedRegistrarServer
	pb.UnimplementedDownstreamServer
	pb.UnimplementedViewerServer

	config utils.ServerConfig

	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup

	// Gathers the known streams
	registrar Registrar

	// Gather the established connections to agents on the field
	agents bags.SortedObj[AgentID, *AgentTwin]
}
