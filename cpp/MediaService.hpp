//
// Created by jfs on 14/01/23.
//

#ifndef CAMS_CPP_MEDIASERVICE_HPP
#define CAMS_CPP_MEDIASERVICE_HPP

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

#endif //CAMS_CPP_MEDIASERVICE_HPP
