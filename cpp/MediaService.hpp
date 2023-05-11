//
// Created by jfs on 14/01/23.
//

#pragma once

#include <grpcpp/grpcpp.h>

#include "hub.pb.h"
#include "hub.grpc.pb.h"
#include "MediaDecoder.hpp"
#include "Uncopyable.hpp"

class MediaService :
        public ::cams::api::hub::Uploader::Service,
        Uncopyable {
public:
    MediaService() : ::cams::api::hub::Uploader::Service() {}

    ~MediaService() override = default;

    grpc::Status MediaUpload(::grpc::ServerContext *context,
                             ::grpc::ServerReader<::cams::api::hub::DownstreamMediaFrame> *stream,
                             ::cams::api::hub::None *response) override;
};
