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

#include "MediaService.hpp"

#include <cstdio>
#include <utility>

#include "AvError.hpp"
#include "FrameSource.hpp"
#include "MediaPipeline.hpp"
#include "StreamStorage.hpp"

namespace {

using ::cams::api::hub::DownstreamMediaFrame;
using ::cams::api::hub::DownstreamMediaFrameType;

// GrpcFrameSource is the live side of the pipeline: libavformat asks for a
// packet and one is read off the upload.
//
// Nothing is queued and no thread is spawned. The demuxer runs on the thread
// serving the call, so a hub that cannot keep up stops reading and the gRPC
// stream applies the back pressure -- which the agent can observe, where a
// queue here could only grow or drop.
class GrpcFrameSource : public FrameSource {
public:
    GrpcFrameSource(::grpc::ServerReader<DownstreamMediaFrame> *stream, uint32_t track)
            : stream_{stream}, track_{track} {}

    int next(uint8_t *buf, int cap) override {
        for (;;) {
            frame_.Clear();
            if (!stream_->Read(&frame_)) {
                // The agent closed its side, or the call was cancelled. Either
                // way the stream is over and the playlist should be finished.
                return AVERROR_EOF;
            }
            // IsInitialized only, and not the IsInitializedWithErrors this
            // code used to also call: that is documented as "identical to
            // IsInitialized() except that it logs an error message", so the
            // condition "!IsInitialized() || IsInitializedWithErrors()" read as
            // "!ok || ok" and rejected every frame ever sent.
            if (!frame_.IsInitialized()) {
                status_ = {grpc::StatusCode::ABORTED, "malformed media frame"};
                return AVERROR_EOF;
            }

            switch (frame_.type()) {
                case DownstreamMediaFrameType::DOWNSTREAM_MEDIA_FRAME_TYPE_RTP:
                case DownstreamMediaFrameType::DOWNSTREAM_MEDIA_FRAME_TYPE_RTCP:
                    break;
                case DownstreamMediaFrameType::DOWNSTREAM_MEDIA_FRAME_TYPE_SDP:
                    // A banner mid-stream would mean a new session, which the
                    // agent opens a new call for.
                    status_ = {grpc::StatusCode::ABORTED,
                               "a second session description on one upload"};
                    return AVERROR_EOF;
                default:
                    status_ = {grpc::StatusCode::ABORTED, "unknown media frame type"};
                    return AVERROR_EOF;
            }

            // The track is why api/hub.proto carries one. The description was
            // reduced to a single media, so a packet from any other would be
            // handed to a demuxer that has no stream for it -- and, with
            // matching done by payload type, might be attributed to the wrong
            // one rather than rejected.
            if (frame_.track() != track_) {
                off_track_++;
                continue;
            }

            const auto &payload = frame_.payload();
            if (payload.size() > static_cast<size_t>(cap)) {
                // Longer than an RTP packet may be; forwarding a prefix would
                // corrupt the stream more quietly than dropping it.
                oversized_++;
                continue;
            }
            if (payload.empty()) {
                empty_++;
                continue;
            }
            memcpy(buf, payload.data(), payload.size());
            return static_cast<int>(payload.size());
        }
    }

    // The status to answer the call with, once the pipeline has unwound.
    [[nodiscard]] const grpc::Status &status() const { return status_; }

    [[nodiscard]] uint64_t off_track() const { return off_track_; }

    [[nodiscard]] uint64_t oversized() const { return oversized_; }

    [[nodiscard]] uint64_t empty() const { return empty_; }

private:
    ::grpc::ServerReader<DownstreamMediaFrame> *stream_;
    uint32_t track_;
    DownstreamMediaFrame frame_;
    grpc::Status status_{grpc::Status::OK};
    uint64_t off_track_{0};
    uint64_t oversized_{0};
    uint64_t empty_{0};
};

} // namespace

MediaService::MediaService(std::string root, HlsOptions opts)
        : ::cams::api::hub::Uploader::Service(),
          root_{std::move(root)},
          opts_{opts} {}

grpc::Status MediaService::MediaUpload(::grpc::ServerContext *context,
                                       ::grpc::ServerReader<DownstreamMediaFrame> *stream,
                                       ::cams::api::hub::None *response) {
    (void) response;

    stream->SendInitialMetadata();

    // 1. Identify the client and the stream from the call's metadata.
    std::string userID, camID, sessionID;
    {
        auto &metadata = context->client_metadata();
        auto it = metadata.find(kMetadataUser);
        if (it == metadata.end()) {
            return {grpc::StatusCode::FAILED_PRECONDITION, "no user id in the \"user\" metadata"};
        }
        userID.assign(it->second.begin(), it->second.end());

        it = metadata.find(kMetadataStream);
        if (it == metadata.end()) {
            return {grpc::StatusCode::FAILED_PRECONDITION, "no stream id in the \"stream\" metadata"};
        }
        camID.assign(it->second.begin(), it->second.end());

        it = metadata.find(kMetadataSession);
        if (it != metadata.end()) {
            sessionID.assign(it->second.begin(), it->second.end());
        }
    }
    if (!stream_id_is_safe(userID) || !stream_id_is_safe(camID)) {
        // These become directory names, so they are checked before use rather
        // than after.
        return {grpc::StatusCode::INVALID_ARGUMENT, "the user or stream id is not a usable name"};
    }

    // 2. Take the stream, displacing whoever holds it. The separator cannot be
    // ambiguous because stream_id_is_safe rejects it in either half. Held by an
    // object rather than released by hand, so every way out of this function --
    // including the error returns below and an exception from libav -- gives it
    // back.
    //
    // The canceller is how a later upload of this camera displaces *this* one:
    // the pipeline is blocked in stream->Read() and only the call's own
    // cancellation will release it.
    const StreamRegistry::Lease lease(streams_, userID + "/" + camID,
                                      [context] { context->TryCancel(); });
    if (!lease.held()) {
        return {grpc::StatusCode::ALREADY_EXISTS,
                "the previous upload of this stream did not yield in time"};
    }

    // 3. Authenticate the stream
    // FIXME(jfs): authenticate the stream

    // 4. Consume the session description that opens the stream.
    DownstreamMediaFrame banner;
    if (!stream->Read(&banner)) {
        return {grpc::StatusCode::OK, "bye"};
    }
    if (!banner.IsInitialized()) {
        return {grpc::StatusCode::ABORTED, "banner read error"};
    }
    if (banner.type() != DownstreamMediaFrameType::DOWNSTREAM_MEDIA_FRAME_TYPE_SDP) {
        return {grpc::StatusCode::ABORTED, "banner expected"};
    }
    const std::string sdp(banner.payload());

    uint32_t track = 0;
    if (!track_of(sdp, &track)) {
        return {grpc::StatusCode::INVALID_ARGUMENT,
                "the session description carries no video"};
    }

    // 5. Run the stream. libavformat pulls from the upload until it ends.
    GrpcFrameSource frames(stream, track);
    StreamStorage storage(root_, userID, camID);
    StreamStats stats;

    const int rc = run_stream(sdp, frames, storage, opts_, &stats);

    fprintf(stderr,
            "cams: stream user=%s cam=%s session=%s track=%u written=%llu skipped=%llu"
            " undated=%llu off_track=%llu oversized=%llu empty=%llu rtcp=%llu\n",
            userID.c_str(), camID.c_str(),
            sessionID.empty() ? "-" : sessionID.c_str(), stats.track,
            static_cast<unsigned long long>(stats.written),
            static_cast<unsigned long long>(stats.skipped),
            static_cast<unsigned long long>(stats.undated),
            static_cast<unsigned long long>(frames.off_track()),
            static_cast<unsigned long long>(frames.oversized()),
            static_cast<unsigned long long>(frames.empty()),
            static_cast<unsigned long long>(stats.rtcp_discarded));

    // A protocol error the source recorded outranks whatever libav made of the
    // truncated stream that followed it.
    if (!frames.status().ok()) {
        return frames.status();
    }
    if (rc < 0) {
        return {grpc::StatusCode::INTERNAL, av_error(rc)};
    }
    return grpc::Status::OK;
}
