// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"github.com/jfsmig/cams/utils"
	"sync"
	"time"

	"github.com/jfsmig/cams/proto"
	"google.golang.org/grpc"
)

type upstreamAgent struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup
}

type connectedUpstreamAgent struct {
	upstream *upstreamAgent

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	cnx *grpc.ClientConn
}

func NewUpstreamAgent(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup) *upstreamAgent {
	return &upstreamAgent{
		ctx:    ctx,
		cancel: cancel,
		wg:     wg,
	}
}

func (us *connectedUpstreamAgent) runRegistration() {
	defer us.wg.Done()
	defer us.cancel()

	ticker := time.Tick(1 * time.Second)

	client := proto.NewRegistrarClient(us.cnx)
	for {
		select {
		case <-us.ctx.Done():
			return
		case <-ticker:
			inReq := proto.RegisterRequest{
				Id: &proto.StreamId{},
			}
			inRep, err := client.Register(us.ctx, &inReq)
			if err != nil {
				utils.Logger.Warn().
					Str("action", "register").
					Err(err).
					Msg("upstream")
				return
			} else {
				utils.Logger.Debug().
					Uint32("status", inRep.Status.Code).
					Str("msg", inRep.Status.Status).
					Str("action", "register").
					Msg("upstream")
			}
		}
	}
	//client.Register(us.cnx)
}

func (us *connectedUpstreamAgent) runStreamCommands() {
	defer us.wg.Done()
	defer us.cancel()

	//client := proto.NewCollectorClient(us.cnx)
	//client.Register(us.cnx)
}

func (us *upstreamAgent) reconnectAndRerun(ctx context.Context, cancel context.CancelFunc, addr string) {
	defer cancel()

	cnx, err := utils.DialGrpc(ctx, addr)
	if err != nil {
		utils.Logger.Error().Err(err).Str("action", "dial").Msg("upstream")
	}
	defer cnx.Close()

	cus := connectedUpstreamAgent{
		upstream: us,
		cnx:      cnx,
		wg:       sync.WaitGroup{},
		ctx:      ctx,
		cancel:   cancel,
	}

	cus.wg.Add(2)
	go cus.runRegistration()
	go cus.runStreamCommands()
	cus.wg.Wait()
}

func (us *upstreamAgent) Run(addr string) {
	defer us.wg.Done()
	defer us.cancel()

	for {
		select {
		case <-us.ctx.Done():
			return
		default:
			<-time.After(time.Second) // Pause to avoid crazy looping of connection attempts
			ctxSub, cancelSub := context.WithCancel(us.ctx)
			us.reconnectAndRerun(ctxSub, cancelSub, addr)
		}
	}
}
