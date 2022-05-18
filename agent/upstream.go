// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"crypto/tls"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/jfsmig/wiy/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"

	"context"
	"sync"
	"time"
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

func dialGrpc(ctx context.Context, endpoint string) (*grpc.ClientConn, error) {
	config := &tls.Config{
		InsecureSkipVerify: true,
	}

	options := []grpc_retry.CallOption{
		grpc_retry.WithCodes(codes.Unavailable),
		grpc_retry.WithBackoff(
			grpc_retry.BackoffExponentialWithJitter(250*time.Millisecond, 0.1),
		),
		grpc_retry.WithMax(5),
		grpc_retry.WithPerRetryTimeout(1 * time.Second),
	}

	return grpc.DialContext(ctx, endpoint,
		grpc.WithTransportCredentials(credentials.NewTLS(config)),
		grpc.WithUnaryInterceptor(
			grpc_middleware.ChainUnaryClient(
				//grpc_prometheus.UnaryClientInterceptor,
				grpc_retry.UnaryClientInterceptor(options...),
			)),
		grpc.WithStreamInterceptor(
			grpc_middleware.ChainStreamClient(
				//grpc_prometheus.StreamClientInterceptor,
				grpc_retry.StreamClientInterceptor(options...),
			)),
	)
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
				Logger.Warn().
					Str("action", "register").
					Err(err).
					Msg("upstream")
				return
			} else {
				Logger.Debug().
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

func (us *upstreamAgent) reconnectAndRerun(ctx context.Context, cancel context.CancelFunc) {
	defer cancel()

	cnx, err := dialGrpc(ctx, "127.0.0.1:6000")
	if err != nil {
		Logger.Error().Err(err).Str("action", "dial").Msg("upstream")
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

func (us *upstreamAgent) Run() {
	defer us.wg.Done()
	defer us.cancel()

	for {
		select {
		case <-us.ctx.Done():
			return
		default:
			<-time.After(time.Second) // Pause to avoid crazy looping of connection attempts
			ctxSub, cancelSub := context.WithCancel(us.ctx)
			us.reconnectAndRerun(ctxSub, cancelSub)
		}
	}
}
