// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"github.com/jfsmig/cams/proto"
	"github.com/jfsmig/cams/utils"
	"github.com/jfsmig/go-bags"
	"net"
	"sync"
)

type StreamID string

type TLSConfig struct {
	PathCrt string `json:"crt"`
	PathKey string `json:"key"`
}

type grpcHub struct {
	proto.UnimplementedRegistrarServer
	proto.UnimplementedControllerServer
	proto.UnimplementedStreamPlayerServer

	config utils.ServerConfig

	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup

	// Gathers the known streams
	registrar Registrar
	player    StreamPlayer

	// Gather the established connections to agent on the field
	agent bags.SortedObj[string, AgentController]
}

func runHub(ctx context.Context, config utils.ServerConfig) error {
	hub := &grpcHub{
		config: config,
	}

	cnx, err := hub.config.ServeTLS()
	if err != nil {
		return err
	}

	listener, err := net.Listen("", hub.config.ListenAddr)
	if err != nil {
		return err
	}

	proto.RegisterRegistrarServer(cnx, hub)
	proto.RegisterControllerServer(cnx, hub)
	proto.RegisterStreamPlayerServer(cnx, hub)

	hub.registrar = NewRegistrarInMem()
	hub.player = NewStreamPlayer()

	return cnx.Serve(listener)
}

func (hub *grpcHub) Register(ctx context.Context, req *proto.RegisterRequest) (*proto.RegisterReply, error) {
	err := hub.registrar.Register(StreamRegistration{req.Id.Stream, req.Id.Stream})
	return &proto.RegisterReply{
		Status: &proto.Status{Code: 202, Status: "registered"},
	}, err
}

func (hub *grpcHub) Control(stream proto.Controller_ControlServer) error {
	agent := NewAgenController(stream)
	return agent.Run()
}

func (hub *grpcHub) Media(stream proto.StreamPlayer_MediaServer) error {
	src := NewStreamSource(stream)
	hub.player.Register(src)
	defer hub.player.Unregister(src)

	return src.Run()
}
