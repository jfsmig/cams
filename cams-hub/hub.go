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

type Hub struct {
	proto.UnimplementedRegistrarServer
	proto.UnimplementedControllerServer

	config utils.ServerConfig

	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup

	// Gathers the known streams
	registrar Registrar

	// Gather the established connections to agent on the field
	agent bags.SortedObj[string, AgentController]
}

type Registrar interface {
	Register(stream StreamRegistration) error

	ListById(start string) ([]StreamRecord, error)
}

type StreamRecord struct {
	StreamID string
	User     string
}

type StreamRegistration struct {
	StreamID string
	User     string
}

func NewHub(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup) *Hub {
	reg := &Hub{
		config: utils.ServerConfig{
			PathCrt: "",
			PathKey: "",
		},
		ctx:    ctx,
		cancel: cancel,
		wg:     wg,
	}
	return reg
}

func (hub *Hub) Run(listenAddr string, reg Registrar) error {
	defer hub.cancel()
	defer hub.wg.Done()

	cnx, err := hub.config.ServeTLS()
	if err != nil {
		return err
	}

	listener, err := net.Listen("", listenAddr)
	if err != nil {
		return err
	}

	proto.RegisterRegistrarServer(cnx, hub)
	proto.RegisterControllerServer(cnx, hub)

	hub.registrar = reg
	return cnx.Serve(listener)
}

func (hub *Hub) Register(ctx context.Context, req *proto.RegisterRequest) (*proto.RegisterReply, error) {
	err := hub.registrar.Register(StreamRegistration{req.Id.Stream, req.Id.Stream})
	return &proto.RegisterReply{
		Status: &proto.Status{Code: 202, Status: "registered"},
	}, err
}

func (hub *Hub) Control(stream proto.Controller_ControlServer) error {
	agent := NewAgenController(stream)
	agent.Run()
	return nil
}

func (hub *Hub) Media(stream proto.Controller_MediaServer) error {
	agent := NewStreamPlayer(stream)
	agent.Run()
	return nil
}
