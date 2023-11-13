// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package utils

import (
	"context"
	"github.com/juju/errors"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"os"
	"strings"
	"time"
)

var (
	// LoggerContext is the builder of a zerolog.Logger that is exposed to the application so that
	// options at the CLI might alter the formatting and the output of the logs.
	LoggerContext = zerolog.
			New(zerolog.ConsoleWriter{
			Out: os.Stderr, TimeFormat: time.RFC3339,
		}).With().Timestamp()

	// Logger is a zerolog logger, that can be safely used from any part of the application.
	// It gathers the format and the output.
	Logger = LoggerContext.Logger()
)

type logEvt struct {
	z     *zerolog.Event
	start time.Time
}

func newEvent(method string) *logEvt {
	return &logEvt{z: Logger.Debug().Str("uri", method), start: time.Now()}
}

func (evt *logEvt) send() { evt.z.Msg("access") }

func (evt *logEvt) setResult(err error) *logEvt {
	evt.z = evt.z.TimeDiff("t", time.Now(), evt.start)
	if err != nil {
		evt.z.Int("rc", 500)
		evt.z.Err(err)
	} else {
		evt.z.Int("rc", 200)
	}
	return evt
}

func (evt *logEvt) patchWithRequest(ctx context.Context) *logEvt {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		auth := md.Get(":authority")
		if len(auth) > 0 {
			evt.z.Str("local", auth[0])
		}
		sessionID := md.Get("session-id")
		if len(sessionID) > 0 {
			evt.z.Str("session", sessionID[0])
		}
	}
	return evt
}

func (evt *logEvt) pathWithReply(ctx context.Context) *logEvt {
	if peer, ok := peer.FromContext(ctx); ok {
		addr := peer.Addr.String()
		if i := strings.LastIndex(addr, ":"); i > -1 {
			addr = addr[:i]
		}
		evt.z.Str("peer", addr)
	}
	return evt
}

func NewStreamServerInterceptorZerolog() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		evt := newEvent(info.FullMethod)
		err := handler(srv, ss)
		ctx := ss.Context()
		evt.setResult(err).patchWithRequest(ctx).pathWithReply(ctx).send()
		return errors.Trace(err)
	}
}

func NewUnaryServerInterceptorZerolog() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		evt := newEvent(info.FullMethod)
		evt.z.Interface("req", req)
		resp, err := handler(ctx, req)
		evt.setResult(err).patchWithRequest(ctx).pathWithReply(ctx).send()
		return resp, err
	}
}
