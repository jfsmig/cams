// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"github.com/jfsmig/cams/api/pb"
	"github.com/jfsmig/cams/utils"
	"github.com/juju/errors"
)

func play(ctx context.Context, address, userID, streamID string) error {
	cnx, err := utils.DialInsecure(ctx, address)
	if err != nil {
		return errors.Annotate(err, "dial")
	}
	defer cnx.Close()

	client := pb.NewViewerClient(cnx)
	_, err = client.Play(ctx, &pb.PlayRequest{
		Id: &pb.StreamId{User: userID, Stream: streamID},
	})
	return err
}
