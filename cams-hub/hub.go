// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"github.com/jfsmig/cams/proto"
	"github.com/jfsmig/cams/utils"
	"github.com/juju/errors"
	"net"
	"sync"
)

type Hub struct {
	proto.UnimplementedRegistrarServer
	proto.UnimplementedCollectorServer

	config utils.ServerConfig

	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup

	registrar Registrar
}

type Registrar interface {
	Register(stream Stream)

	ListById(start string)
}

type Stream struct {
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

func (reg *Hub) Run(listenAddr string) error {
	defer reg.cancel()
	defer reg.wg.Done()

	cnx, err := reg.config.ServeTLS()
	if err != nil {
		return err
	}

	listener, err := net.Listen("", listenAddr)
	if err != nil {
		return err
	}

	proto.RegisterRegistrarServer(cnx, reg)
	proto.RegisterCollectorServer(cnx, reg)

	return cnx.Serve(listener)
}

func (reg *Hub) Register(ctx context.Context, req *proto.RegisterRequest) (*proto.RegisterReply, error) {
	return nil, errors.NotImplemented
}

func (reg *Hub) Play(stream proto.Collector_PlayServer) error {
	return errors.NotImplemented
}

func (reg *Hub) Pause(stream proto.Collector_PauseServer) error {
	return errors.NotImplemented
}
