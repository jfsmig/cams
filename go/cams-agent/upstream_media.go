package main

import (
	"context"

	"github.com/aler9/gortsplib/v2/pkg/format"
	"github.com/aler9/gortsplib/v2/pkg/media"
	"github.com/jfsmig/cams/go/api/pb"
	"github.com/jfsmig/cams/go/camera"
	"github.com/jfsmig/cams/go/utils"
	"github.com/juju/errors"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type grpcUpstream struct {
	cnx          *grpc.ClientConn
	uploadClient pb.Uploader_MediaUploadClient
}

func (gu *grpcUpstream) Close() {
	_ = gu.cnx.Close()
}

func (gu *grpcUpstream) OnSDP(sdp string) error {
	frame := &pb.DownstreamMediaFrame{
		Type:    pb.DownstreamMediaFrameType_DOWNSTREAM_MEDIA_FRAME_TYPE_SDP,
		Payload: []byte(sdp),
	}
	return gu.uploadClient.Send(frame)
}

func (gu *grpcUpstream) OnRTP(m *media.Media, f format.Format, pkt *rtp.Packet) error {
	b, err := pkt.Marshal()
	if err == nil {
		frame := &pb.DownstreamMediaFrame{
			Type:    pb.DownstreamMediaFrameType_DOWNSTREAM_MEDIA_FRAME_TYPE_RTP,
			Payload: b,
		}
		err = gu.uploadClient.Send(frame)
	}
	return err
}

func (gu *grpcUpstream) OnRTCP(m *media.Media, pkt *rtcp.Packet) error {
	b, err := (*pkt).Marshal()
	if err == nil {
		frame := &pb.DownstreamMediaFrame{
			Type:    pb.DownstreamMediaFrameType_DOWNSTREAM_MEDIA_FRAME_TYPE_RTCP,
			Payload: b,
		}
		err = gu.uploadClient.Send(frame)
	}
	return err
}

func NewGrpcUploadMaker(userID, camID, url string) camera.UploadOpenFunc {
	return func(ctx context.Context) (camera.UpstreamMedia, error) {
		var err error
		up := &grpcUpstream{}
		up.cnx, err = utils.DialInsecure(ctx, url)
		if err != nil {
			return nil, errors.Annotate(err, "dial")
		}

		client := pb.NewUploaderClient(up.cnx)
		ctx = metadata.AppendToOutgoingContext(ctx,
			utils.KeyUser, userID,
			utils.KeyStream, camID)
		up.uploadClient, err = client.MediaUpload(ctx)
		if err != nil {
			up.Close()
			return nil, errors.Annotate(err, "call")
		}

		return up, nil
	}
}
