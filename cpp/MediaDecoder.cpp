//
// Created by jfs on 14/01/23.
//

#include "MediaDecoder.hpp"

#include <cstdio>
#include <cassert>

#include <list>
#include <iostream>

#include "RTP.hpp"
#include "NAL.hpp"

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
    auto pkt = RtpPacket::parse(buf, len);
#if 0
    ::fprintf(stderr,"  Header:"
             " v=%01x padding=%01x ext=%01x crsrc_count=%01x"
             " marker=%01x payload_type=%01x"
             " sequence_number=%02x timestamp=%04x ssrc_id=%04x\n",
             pkt.header.version,
             pkt.header.padding,
             pkt.header.extension,
             pkt.header.crsc_count,
             pkt.header.marker,
             pkt.header.payload_type,
             pkt.header.sequence_number,
             pkt.header.timestamp,
             pkt.header.ssrc_id);
    for (int i=0; i<pkt.header.crsc_count; i++)
        ::fprintf(stderr, "  CSRC: %04x\n", pkt.csrc_id[i]);
    if (pkt.header.extension)
        ::fprintf(stderr, "  EXT: id=%02x length=%02x\n",
                  pkt.extension.preamble.id, pkt.extension.preamble.length);

    ::fprintf(stderr, "  DATA:\n");
    hex(pkt.payload, 32);

    ::fprintf(stderr, " FRAME:\n");
    hex(buf, len);
#endif
    if (pkt.payload_size <= 2)
        return false;

    auto nalus = parse_train(pkt.payload, pkt.payload_size);
    for (const auto &nalu: nalus) {
        ::fprintf(stderr, "  NALU: forbidden=%01x idc=%01x type=%02x length=%lu",
                  nalu.header.forbidden,
                  nalu.header.idc,
                  nalu.header.type,
                  nalu.length);
        // TODO(jfs): extract the frames from the decode once fed with the NALU
        MediaFrame frame;
        if (!encoder_.on_frame(frame))
            return false;
    }
    return true;
}
