//
// Created by jfs on 14/01/23.
//

#ifndef CAMS_CPP_MEDIASERVICE_HPP
#define CAMS_CPP_MEDIASERVICE_HPP

#include <grpcpp/grpcpp.h>

#include "hub.pb.h"
#include "hub.grpc.pb.h"
#include "MediaDecoder.hpp"

class MediaService : public ::cams::api::hub::Uploader::Service {
public:
    MediaService() : ::cams::api::hub::Uploader::Service() {};

    ~MediaService() override = default;

    grpc::Status MediaUpload(::grpc::ServerContext *context,
                             ::grpc::ServerReader<::cams::api::hub::DownstreamMediaFrame> *stream,
                             ::cams::api::hub::None *response) override {
        ::cams::api::hub::DownstreamMediaFrame frame;

        stream->SendInitialMetadata();

        // 1. Authenticate the client and the stream using the fields present in the metadata
        std::string userID, camID;
        {
            auto &metadata = context->client_metadata();
            auto it = metadata.find("user");
            if (it == metadata.end()) {
                return {grpc::StatusCode::FAILED_PRECONDITION, "no user id"};
            } else {
                userID.assign(it->second.begin(), it->second.end());
            }
            it = metadata.find("camera");
            if (it == metadata.end()) {
                return {grpc::StatusCode::FAILED_PRECONDITION, "no camera id"};
            } else {
                camID.assign(it->second.begin(), it->second.end());
            }
        }

        // 2. Authenticate the stream
        // FIXME(jfs): authenticate the stream

        // 3. Consume the SDP banner that describes the stream format
        if (!stream->Read(&frame)) {
            return {grpc::StatusCode::OK, "bye"};
        }
        if (!frame.IsInitialized() || frame.IsInitializedWithErrors()) {
            return {grpc::StatusCode::ABORTED, "banner read error"};
        }
        if (frame.type() != ::cams::api::hub::DownstreamMediaFrameType::DOWNSTREAM_MEDIA_FRAME_TYPE_SDP) {
            return {grpc::StatusCode::ABORTED, "banner expected"};
        }

        // 4. Everything is know, then prepare the pipeline
        StreamStorage storage(userID, camID);
        MediaEncoder encoder(storage);
        MediaDecoder output(std::string_view(frame.payload().data(), frame.payload.size()));

        // 5. Loop on the RTP/RTCP frames to feed the pipeline
        for (;;) {
            frame.Clear();
            if (!stream->Read(&frame)) {
                return {grpc::StatusCode::OK, "bye"};
            }
            if (!frame.IsInitialized() || frame.IsInitializedWithErrors()) {
                return {grpc::StatusCode::ABORTED, "payload read error"};
            }
            switch (frame.type()) {
                case ::cams::api::hub::DownstreamMediaFrameType::DOWNSTREAM_MEDIA_FRAME_TYPE_SDP:
                    return {grpc::StatusCode::ABORTED, "unexpected payload frame"};
                case ::cams::api::hub::DownstreamMediaFrameType::DOWNSTREAM_MEDIA_FRAME_TYPE_RTP:
                    output.onRTP(frame.payload().data(), frame.payload().size());
                    continue;
                case ::cams::api::hub::DownstreamMediaFrameType::DOWNSTREAM_MEDIA_FRAME_TYPE_RTCP:
                    output.onRTCP(frame.payload().data(), frame.payload().size());
                    continue;
            }
        }
    }
};

#endif //CAMS_CPP_MEDIASERVICE_HPP
