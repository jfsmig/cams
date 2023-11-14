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
	"github.com/jfsmig/cams/go/utils"
	"github.com/juju/errors"
)

func hubPlay(ctx context.Context, address, userID, streamID string) error {
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
