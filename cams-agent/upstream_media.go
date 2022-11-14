// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"github.com/jfsmig/cams/api/pb"
	"github.com/jfsmig/cams/defs"
	"github.com/jfsmig/cams/utils"
	"go.nanomsg.org/mangos/v3/protocol"
	"go.nanomsg.org/mangos/v3/protocol/pull"
	"google.golang.org/grpc"
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
	control       chan UpstreamMediaCommand
	singletonLock sync.Mutex
}

func NewUpstreamMedia(camID string) UpstreamMedia {
	return &upstreamMedia{
		camID:         camID,
		control:       make(chan UpstreamMediaCommand, 4),
		singletonLock: sync.Mutex{},
	}
}

func (um *upstreamMedia) PK() string    { return um.camID }
func (um *upstreamMedia) CommandPlay()  { um.control <- upstreamMedia_CommandPlay }
func (um *upstreamMedia) CommandPause() { um.control <- upstreamMedia_CommandPause }
func (um *upstreamMedia) CommandShut()  { um.control <- upstreamMedia_CommandShut }

func (um *upstreamMedia) Run(ctx context.Context, cnx *grpc.ClientConn) {
	ctx = context.WithValue(ctx, defs.KeyStream, um.camID)

	utils.Logger.Trace().Str("action", "start").Str("camera", um.camID).Msg("upstream media")

	if !um.singletonLock.TryLock() {
		panic("BUG the upstream media is already running")
	}
	defer um.singletonLock.Unlock()

	// Connect to the internal media bridge fed by the camera
	socketRtp, err := pull.NewSocket()
	if err != nil {
		utils.Logger.Warn().Str("action", "north socket").Err(err).Msg("upstream media")
		return
	}
	defer func() { _ = socketRtp.Close() }()
	if err = socketRtp.Dial(makeNorthRtp(um.camID)); err != nil {
		utils.Logger.Warn().Str("action", "north dial").Err(err).Msg("upstream media")
		return
	}

	// Connect to the internal media bridge fed by the camera
	socketRtcp, err := pull.NewSocket()
	if err != nil {
		utils.Logger.Warn().Str("action", "north socket").Err(err).Msg("upstream media")
		return
	}
	defer func() { _ = socketRtcp.Close() }()
	if err = socketRtcp.Dial(makeNorthRtcp(um.camID)); err != nil {
		utils.Logger.Warn().Str("action", "north dial").Err(err).Msg("upstream media")
		return
	}

	// Open a gRPC connection for the upstream
	client := pb.NewControllerClient(cnx)
	ctrl, err := client.MediaUpload(ctx)
	if err != nil {
		utils.Logger.Warn().Str("action", "open").Err(err).Msg("upstream media")
		return
	}

	utils.GroupRun(ctx,
		func(c context.Context) { upstreamPipeline(c, socketRtp, ctrl, pb.MediaFrameType_FrameType_RTP) },
		func(c context.Context) { upstreamPipeline(c, socketRtcp, ctrl, pb.MediaFrameType_FrameType_RTCP) },
	)
}

func upstreamPipeline(ctx context.Context, sock protocol.Socket, ctrl pb.Controller_MediaUploadClient, frameType pb.MediaFrameType) {
	proto := "rtp"
	if frameType == pb.MediaFrameType_FrameType_RTCP {
		proto = "rtcp"
	}

	for {
		if err := ctx.Err(); err != nil {
			utils.Logger.Warn().Str("action", "interrupted").Err(err).Str("proto", proto).Msg("upstream media")
			return
		}
		msg, err := sock.Recv()
		if err != nil {
			utils.Logger.Warn().Str("action", "consume").Err(err).Msg("upstream media")
			return
		}

		frame := &pb.MediaFrame{
			Type:    frameType,
			Payload: msg,
		}
		if err := ctrl.Send(frame); err != nil {
			utils.Logger.Warn().Str("action", "send").Err(err).Msg("upstream media")
			return
		}
	}
}
