//
// Created by jfs on 14/01/23.
//

#include "MediaDecoder.hpp"

#include <cassert>
#include <iostream>

#include "RTP.hpp"

// https://blog.kevmo314.com/custom-rtp-io-with-ffmpeg.html
// https://stackoverflow.com/questions/71000853/demuxing-and-decoding-raw-rtp-with-libavformat
// https://gist.github.com/jl2/1681387

MediaDecoder::MediaDecoder(const std::string_view sdp,
                           MediaEncoder &encoder) : encoder_{encoder} {
    // TODO extract this from the SDP description
    AVCodecID codec_video_id = AV_CODEC_ID_H264, codec_audio_id = AV_CODEC_ID_MPEG4;

    input_format_context_ = avformat_alloc_context();
    assert(input_format_context_ != nullptr);

    input_format_context_->bit_rate = 90000;

    input_format_context_->audio_codec_id = codec_audio_id;
    input_format_context_->audio_codec = avcodec_find_decoder(codec_audio_id);
    assert(input_format_context_->audio_codec != nullptr);
    audio_codec_ctx_ = avcodec_alloc_context3(input_format_context_->audio_codec);
    assert(audio_codec_ctx_ != nullptr);

    input_format_context_->video_codec_id = codec_audio_id;
    input_format_context_->video_codec = avcodec_find_decoder(codec_video_id);
    assert(input_format_context_->video_codec != nullptr);
    video_codec_ctx_ = avcodec_alloc_context3(input_format_context_->video_codec);
    assert(video_codec_ctx_ != nullptr);

    int rc;
    rc = avcodec_open2(video_codec_ctx_, input_format_context_->video_codec, nullptr);
    assert(rc == 0);
    rc = avcodec_open2(audio_codec_ctx_, input_format_context_->audio_codec, nullptr);
    assert(rc == 0);

    frame_in_ = av_frame_alloc();
    assert(frame_in_ != nullptr);
    frame_out_ = av_frame_alloc();
    assert(frame_out_ != nullptr);
}

MediaDecoder::~MediaDecoder() {
    av_frame_free(&frame_in_);
    av_frame_free(&frame_out_);

    avcodec_close(video_codec_ctx_);
    avcodec_close(audio_codec_ctx_);

    avformat_close_input(&input_format_context_);
    avformat_free_context(input_format_context_);
}

bool MediaDecoder::on_rtp(const uint8_t *buf, size_t len) {
    auto header = RtpHeader<1>::parse(buf);

    ::fprintf(stderr, "  Header:"
                " v=%01x padding=%01x ext=%01x crsrc_count=%01x"
             " marker=%01x payload_type=%01x"
             " sequence_number=%02x timestamp=%04x ssrc_id=%04x\n",
             header.header.version,
             header.header.padding,
             header.header.extension,
             header.header.crsc_count,
             header.header.marker,
             header.header.payload_type,
             header.header.sequence_number,
             header.header.timestamp,
             header.header.ssrc_id);

    return encoder_.on_frame(buf, len);
}
