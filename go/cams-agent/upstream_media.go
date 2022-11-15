// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	pb2 "github.com/jfsmig/cams/go/api/pb"
	utils2 "github.com/jfsmig/cams/go/utils"
	"go.nanomsg.org/mangos/v3/protocol"
	"go.nanomsg.org/mangos/v3/protocol/pull"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"sync"
)

type UpstreamMediaCommand string

const (
	upstreamMedia_CommandPlay  UpstreamMediaCommand = "1"
	upstreamMedia_CommandPause                      = "0"
	upstreamMedia_CommandShut                       = "X"
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
	utils2.Logger.Trace().Str("action", "start").Str("camera", um.camID).Msg("upstream media")

	if !um.singletonLock.TryLock() {
		panic("BUG the upstream media is already running")
	}
	defer um.singletonLock.Unlock()

	// Connect to the internal media bridge fed by the camera
	socketRtp, err := pull.NewSocket()
	if err != nil {
		utils2.Logger.Warn().Str("action", "north socket").Err(err).Msg("upstream media")
		return
	}
	defer func() { _ = socketRtp.Close() }()
	if err = socketRtp.Dial(makeNorthRtp(um.camID)); err != nil {
		utils2.Logger.Warn().Str("action", "north dial").Err(err).Msg("upstream media")
		return
	}

	// Connect to the internal media bridge fed by the camera
	socketRtcp, err := pull.NewSocket()
	if err != nil {
		utils2.Logger.Warn().Str("action", "north socket").Err(err).Msg("upstream media")
		return
	}
	defer func() { _ = socketRtcp.Close() }()
	if err = socketRtcp.Dial(makeNorthRtcp(um.camID)); err != nil {
		utils2.Logger.Warn().Str("action", "north dial").Err(err).Msg("upstream media")
		return
	}

	// Open a gRPC connection for the upstream
	client := pb2.NewControllerClient(cnx)

	ctx = metadata.AppendToOutgoingContext(ctx,
		utils2.KeyUser, um.cfg.User,
		utils2.KeyStream, um.camID)

	ctrl, err := client.MediaUpload(ctx)
	if err != nil {
		utils2.Logger.Warn().Str("action", "open").Err(err).Msg("upstream media")
		return
	}

	utils2.GroupRun(ctx,
		func(c context.Context) { upstreamPipeline(c, socketRtp, ctrl, pb2.MediaFrameType_FrameType_RTP) },
		func(c context.Context) { upstreamPipeline(c, socketRtcp, ctrl, pb2.MediaFrameType_FrameType_RTCP) },
	)
}

func upstreamPipeline(ctx context.Context, sock protocol.Socket, ctrl pb2.Controller_MediaUploadClient, frameType pb2.MediaFrameType) {
	proto := "rtp"
	if frameType == pb2.MediaFrameType_FrameType_RTCP {
		proto = "rtcp"
	}

	for {
		if err := ctx.Err(); err != nil {
			utils2.Logger.Warn().Str("action", "interrupted").Err(err).Str("proto", proto).Msg("upstream media")
			return
		}
		msg, err := sock.Recv()
		if err != nil {
			utils2.Logger.Warn().Str("action", "consume").Err(err).Msg("upstream media")
			return
		}

		frame := &pb2.MediaFrame{
			Type:    frameType,
			Payload: msg,
		}
		if err := ctrl.Send(frame); err != nil {
			utils2.Logger.Warn().Str("action", "send").Err(err).Msg("upstream media")
			return
		}
	}
}
