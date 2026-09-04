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

//
// The hub's media plane.
//

#pragma once

#include <string>

#include <grpcpp/grpcpp.h>

#include "hub.pb.h"
#include "hub.grpc.pb.h"
#include "FragmentSink.hpp"
#include "StreamRegistry.hpp"
#include "Uncopyable.hpp"

// The gRPC metadata keys the agent tags a media upload with.
//
// The wire contract belongs to api/hub.proto and go/utils/constants.go, which
// are the source of truth (see AGENTS.md); these mirror utils.KeyUser and
// utils.KeyStream and must not diverge from them. They were "user" and
// "camera" here, so every upload was refused: the agent sends "stream".
constexpr char kMetadataUser[] = "user";
constexpr char kMetadataStream[] = "stream";

// The agent's run, mirroring utils.KeySession. Optional, unlike the two above:
// an agent from before it existed sends none, and a stream is not worth
// refusing over a log field. It is diagnostic only -- it joins this log to the
// agent's and the control plane's, and it is asserted by the caller, so it may
// never be used to decide anything.
constexpr char kMetadataSession[] = "session-id";

class MediaService :
        public ::cams::api::hub::Uploader::Service,
        Uncopyable {
public:
    MediaService() = delete;

    // root is the directory each stream's playlist and segments go under.
    MediaService(std::string root, HlsOptions opts);

    ~MediaService() override = default;

    grpc::Status MediaUpload(::grpc::ServerContext *context,
                             ::grpc::ServerReader<::cams::api::hub::DownstreamMediaFrame> *stream,
                             ::cams::api::hub::None *response) override;

private:
    std::string root_;
    HlsOptions opts_;

    // One upload at a time per (user, camera); see StreamRegistry for why the
    // newcomer displaces the incumbent rather than being refused.
    StreamRegistry streams_;
};
