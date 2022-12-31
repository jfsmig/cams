// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"sync"
	"time"

	"go.nanomsg.org/mangos/v3/protocol"
	"go.nanomsg.org/mangos/v3/protocol/pull"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/jfsmig/cams/go/api/pb"
	"github.com/jfsmig/cams/go/utils"
)

type UpstreamMediaCommand uint32

const (
	upstreamMedia_CommandPlay UpstreamMediaCommand = iota
	upstreamMedia_CommandPause
	upstreamMedia_CommandShut
)

type UpstreamMedia interface {
	PK() string
	Run(ctx context.Context, cnx *grpc.ClientConn)

	CommandPlay()
	CommandPause()
	CommandShut()
}

type upstreamMedia struct {
	camID         string
	cfg           AgentConfig
	control       chan UpstreamMediaCommand
	singletonLock sync.Mutex
}

func NewUpstreamMedia(camID string, cfg AgentConfig) UpstreamMedia {
	return &upstreamMedia{
		camID:         camID,
		cfg:           cfg,
		control:       make(chan UpstreamMediaCommand, 4),
		singletonLock: sync.Mutex{},
	}
}

func (um *upstreamMedia) PK() string    { return um.camID }
func (um *upstreamMedia) CommandPlay()  { um.control <- upstreamMedia_CommandPlay }
func (um *upstreamMedia) CommandPause() { um.control <- upstreamMedia_CommandPause }
func (um *upstreamMedia) CommandShut()  { um.control <- upstreamMedia_CommandShut }

func (um *upstreamMedia) Run(ctx context.Context, cnx *grpc.ClientConn) {
	utils.Logger.Trace().Str("action", "start").Str("camera", um.camID).Msg("up media")

	if !um.singletonLock.TryLock() {
		panic("BUG the up media is already running")
	}
	defer um.singletonLock.Unlock()

	// Connect to the internal media bridge fed by the camera
	socketRtp, err := pull.NewSocket()
	if err != nil {
		utils.Logger.Warn().Str("action", "north socket").Err(err).Msg("up media")
		return
	}
	// Connect to the internal media bridge fed by the camera
	socketRtcp, err := pull.NewSocket()
	if err != nil {
		utils.Logger.Warn().Str("action", "north socket").Err(err).Msg("up media")
		return
	}
	defer func() { _ = socketRtcp.Close() }()

	// Try to connect to the internal media bridge fed by the camera, until it either works or get cancelled
	for ctx.Err() == nil {
		if err = socketRtp.Dial(makeNorthRtp(um.camID)); err != nil {
			utils.Logger.Warn().Str("action", "north rtp dial").Err(err).Msg("up media")
			time.Sleep(200 * time.Millisecond)
		} else {
			break
		}
	}
	for ctx.Err() == nil {
		if err = socketRtcp.Dial(makeNorthRtcp(um.camID)); err != nil {
			utils.Logger.Warn().Str("action", "north rtcp dial").Err(err).Msg("up media")
			time.Sleep(200 * time.Millisecond)
		} else {
			break
		}
	}
	if ctx.Err() != nil {
		return
	} else {
		utils.Logger.Trace().Str("action", "north dial").Msg("up media")
	}

	// Open a gRPC connection for the upstream
	client := pb.NewDownstreamClient(cnx)

	ctx = metadata.AppendToOutgoingContext(ctx,
		utils.KeyUser, um.cfg.User,
		utils.KeyStream, um.camID)

	var ctrl pb.Downstream_MediaUploadClient
	for ctx.Err() == nil {
		ctrl, err = client.MediaUpload(ctx)
		if err != nil {
			utils.Logger.Warn().Str("action", "grpc dial").Err(err).Msg("up media")
			time.Sleep(500 * time.Millisecond)
		} else {
			break
		}
	}
	if ctx.Err() != nil {
		return
	} else {
		utils.Logger.Trace().Str("action", "grpc dial").Msg("up media")
	}

	utils.GroupRun(ctx,
		func(c context.Context) {
			upstreamPipeline(c, socketRtp, ctrl, pb.DownstreamMediaFrameType_DOWNSTREAM_MEDIA_FRAME_TYPE_RTP)
		},
		func(c context.Context) {
			upstreamPipeline(c, socketRtcp, ctrl, pb.DownstreamMediaFrameType_DOWNSTREAM_MEDIA_FRAME_TYPE_RTCP)
		},
	)
}

func upstreamPipeline(ctx context.Context, sock protocol.Socket, ctrl pb.Downstream_MediaUploadClient, frameType pb.DownstreamMediaFrameType) {
	proto := "rtp"
	if frameType == pb.DownstreamMediaFrameType_DOWNSTREAM_MEDIA_FRAME_TYPE_RTCP {
		proto = "rtcp"
	}

	for {
		if err := ctx.Err(); err != nil {
			utils.Logger.Warn().Str("action", "interrupted").Err(err).Str("proto", proto).Msg("up media")
			return
		}
		msg, err := sock.Recv()
		if err != nil {
			utils.Logger.Warn().Str("action", "consume").Err(err).Msg("up media")
			return
		}

		frame := &pb.DownstreamMediaFrame{
			Type:    frameType,
			Payload: msg,
		}
		if err := ctrl.Send(frame); err != nil {
			utils.Logger.Warn().Str("action", "send").Err(err).Msg("up media")
			return
		}
	}
}
