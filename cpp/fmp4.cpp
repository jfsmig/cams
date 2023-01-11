//
// Created by jfs on 09/01/23.
//

#include <cassert>
#include <cstdint>
#include <iostream>
#include <array>

#include <libavformat/avformat.h>
#include <libavcodec/avcodec.h>
#include <libavutil/opt.h>

extern "C" int read_packet(void *opaque, uint8_t *buf, int buf_size) {}

extern "C" int write_packet(void *opaque, uint8_t *buf, int buf_size) {}

extern "C" int64_t seek(void *opaque, int64_t offset, int whence) {}

int main(int argc, char *argv[]) {
    (void) argc, (void) argv;
    const char *filename = "testvideo.fmp4";

    // https://blog.kevmo314.com/custom-rtp-io-with-ffmpeg.html
    // https://stackoverflow.com/questions/71000853/demuxing-and-decoding-raw-rtp-with-libavformat

    std::string sdp = R"(v=0
o=- 0 0 IN IP4 127.0.0.1
s=No Name
c=IN IP4 127.0.0.1
t=0 0
a=tool:libavformat 58.29.100
m=video 5000 RTP/AVP 96
a=rtpmap:96 H264/90000
a=fmtp:96 level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42001f)";

    AVFormatContext *input_format_context = avformat_alloc_context();
    assert(input_format_context != nullptr);

    AVInputFormat *file_iformat = av_find_input_format("sdp");
    AVDictionary *format_opts = nullptr;
    av_dict_set(&format_opts, "sdp_flags", "custom_io", 0);
    auto rc = avformat_open_input(&input_format_context, "video.sdp", file_iformat, &format_opts);
    assert(rc == 0);

    std::cout
        << "format " << input_format_context->iformat->name
        << "duration " << input_format_context->duration
        << "bitrate " << input_format_context->bit_rate
        << std::endl;

    std::array<uint8_t, 8192> readbuf;
    AVIOContext * avio_in = avio_alloc_context(
            readbuf.data(), readbuf.size(),1,
            nullptr, &read_packet, &write_packet, NULL);
    
    avformat_close_input(&input_format_context);
    avformat_free_context(input_format_context);

    {
        AVFormatContext *avfc{nullptr};
        avformat_alloc_output_context2(&avfc, NULL, NULL, filename);
        {
            AVStream *stream = avformat_new_stream(avfc, NULL);
            {
                AVCodec *h264 = avcodec_find_encoder(AV_CODEC_ID_H264);
                AVCodecContext *avcc = avcodec_alloc_context3(h264);

                av_opt_set(avcc->priv_data, "preset", "fast", 0);
                av_opt_set(avcc->priv_data, "crf", "20", 0);
                avcc->thread_count = 1;
                avcc->width = 1920;
                avcc->height = 1080;
                avcc->pix_fmt = AV_PIX_FMT_YUV420P;
                avcc->time_base = av_make_q(1, 5000);
                stream->time_base = avcc->time_base;
                if (avfc->oformat->flags & AVFMT_GLOBALHEADER)
                    avcc->flags |= AV_CODEC_FLAG_GLOBAL_HEADER;

                avcodec_open2(avcc, h264, NULL);
                avcodec_parameters_from_context(stream->codecpar, avcc);
                {

                    avio_open(&avfc->pb, filename, AVIO_FLAG_WRITE);
                    
                }
                avcodec_free_context(&avcc);
            }

        }
        avformat_free_context(avfc);
    }
    return 0;
}