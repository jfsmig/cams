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
	"github.com/jfsmig/cams/go/api/pb"
	"github.com/jfsmig/cams/go/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MediaUpload is the hub's data plane, and this Go hub does not implement it:
// the depacketizing and the HLS storage live in cpp/hub.
//
// It refuses cleanly rather than accepting frames it would drop, so that an
// agent learns at once that this endpoint carries no media. It used to
// panic("implement me"), on a stream any agent opens as soon as a camera plays
// — with no recover anywhere, the first frame took the whole hub down and the
// control plane with it.
func (hub *grpcHub) MediaUpload(server pb.Uploader_MediaUploadServer) error {
	err := status.Error(codes.Unimplemented, "this hub carries no media, see cpp/hub")
	utils.Logger.Warn().Err(err).Str("action", "upload").Msg("hub")
	return err
}
