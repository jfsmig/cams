// Copyright (c) 2022-2024 The authors (see the AUTHORS file)
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"context"

	"github.com/jfsmig/cams/go/api/pb"
	"github.com/jfsmig/cams/go/camera"
	"github.com/jfsmig/cams/go/utils"
	"github.com/juju/errors"
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

func (gu *grpcUpstream) OnRTP(pkt []byte) error {
	frame := &pb.DownstreamMediaFrame{
		Type:    pb.DownstreamMediaFrameType_DOWNSTREAM_MEDIA_FRAME_TYPE_RTP,
		Payload: pkt,
	}
	return gu.uploadClient.Send(frame)
}

func (gu *grpcUpstream) OnRTCP(pkt []byte) error {
	frame := &pb.DownstreamMediaFrame{
		Type:    pb.DownstreamMediaFrameType_DOWNSTREAM_MEDIA_FRAME_TYPE_RTCP,
		Payload: pkt,
	}
	return gu.uploadClient.Send(frame)
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
