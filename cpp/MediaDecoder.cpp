//
// Created by jfs on 14/01/23.
//

#include "MediaDecoder.hpp"

#include <cassert>

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

    input_format_context_->video_codec_id = codec_audio_id;
    input_format_context_->video_codec = avcodec_find_decoder(codec_video_id);
    assert(input_format_context_->video_codec != nullptr);
}

MediaDecoder::~MediaDecoder() {
    avformat_close_input(&input_format_context_);
    avformat_free_context(input_format_context_);
}

bool MediaDecoder::on_rtp(const uint8_t *buf, size_t len) {
    auto header = RtpHeader<1>::parse(buf);
    return encoder_.on_frame(buf, len);
}
