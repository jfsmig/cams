//
// Created by jfs on 14/01/23.
//

#include "hub.pb.h"
#include "hub.grpc.pb.h"
#include "MediaService.hpp"
#include "MediaSource.hpp"

class GrpcSource : public MediaSource {
public:
    GrpcSource() = delete;
    explicit GrpcSource(::grpc::ServerReader<::cams::api::hub::DownstreamMediaFrame> *in) : input_{in} {}

    int Read(uint8_t *buf, size_t buf_size) override {
        ::cams::api::hub::DownstreamMediaFrame frame;

        label_retry:
        frame.Clear();
        if (!input_->Read(&frame)) {
            // TODO(jfs) log
            return AVERROR_EXIT;
        }
        if (!frame.IsInitialized() || frame.IsInitializedWithErrors()) {
            // TODO(jfs) log
            return AVERROR_EXIT;
        }

        switch (frame.type()) {
            case ::cams::api::hub::DownstreamMediaFrameType::DOWNSTREAM_MEDIA_FRAME_TYPE_SDP:
                return -1;
            case ::cams::api::hub::DownstreamMediaFrameType::DOWNSTREAM_MEDIA_FRAME_TYPE_RTCP:
                // TODO(jfs) log
                goto label_retry;
            case ::cams::api::hub::DownstreamMediaFrameType::DOWNSTREAM_MEDIA_FRAME_TYPE_RTP:
                assert(buf_size > 0);
                if (frame.payload().size() > (size_t) buf_size) {
                    // TODO(jfs) log
                    return AVERROR_BUFFER_TOO_SMALL;
                }
                memcpy(buf, frame.payload().data(), frame.payload().size());
                return frame.payload().size();;
            default:
                // TODO(jfs) log
                return AVERROR_BUG;
        }
    }
private:
    ::grpc::ServerReader<::cams::api::hub::DownstreamMediaFrame> *input_ = nullptr;
};

grpc::Status MediaService::MediaUpload(::grpc::ServerContext *context,
                                       ::grpc::ServerReader<::cams::api::hub::DownstreamMediaFrame> *stream,
                                       ::cams::api::hub::None *response) {
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
    std::string sdp(frame.payload());
    StreamStorage storage(userID, camID);
    MediaEncoder encoder(storage);
    GrpcSource source(stream);
    MediaDecoder output(sdp, source, encoder);

    return {grpc::StatusCode::OK, "uploaded"};
}
