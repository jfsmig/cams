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
// cams-rtp2hls: the hub's media plane. It accepts the agents' RTP uploads and
// writes each camera's stream out as HLS.
//

#include <charconv>
#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <string>
#include <system_error>

#include <grpcpp/grpcpp.h>

extern "C" {
#include <libavformat/avformat.h>
#include <libavutil/log.h>
}

#include "hub.pb.h"
#include "hub.grpc.pb.h"
#include "MediaService.hpp"

namespace {

constexpr char kDefaultListen[] = "127.0.0.1:6001";
constexpr char kDefaultRoot[] = "/var/lib/cams/hls";

// out is stdout when the help was asked for and stderr when it is a diagnostic:
// help that was requested is the tool's output, and belongs where a pipe can
// read it.
void usage(FILE *out, const char *argv0) {
    fprintf(out,
            "usage: %s [options]\n"
            "  --listen ADDR   where to accept uploads (default %s)\n"
            "  --root DIR      where to write the playlists (default %s)\n"
            "  --segment SECS  target fragment length (default 4)\n"
            "  --retention S   how far back the playlist reaches, in seconds\n"
            "                  (default 1800); fragments past it are deleted\n"
            "  --keep-all      keep and list every fragment, unbounded\n"
            "  --verbose       raise the libav log level\n",
            argv0, kDefaultListen, kDefaultRoot);
}

// A missing value would otherwise read the terminating null as the argument.
bool value_of(int argc, char **argv, int i, const char **out) {
    if (i + 1 >= argc) {
        fprintf(stderr, "%s needs a value\n", argv[i]);
        return false;
    }
    *out = argv[i + 1];
    return true;
}

// positive_seconds parses a duration, rejecting what atoi used to accept
// silently: atoi answers 0 for "abc" and 4 for "4x", so a typo in a flag became
// a value nobody chose. from_chars reports both the junk and the overflow, and
// the message names the flag and quotes what arrived.
bool positive_seconds(const char *flag, const char *value, int *out) {
    const char *end = value + strlen(value);
    int parsed{0};
    const auto [stop, ec] = std::from_chars(value, end, parsed);
    if (ec == std::errc{} && stop == end && parsed > 0) {
        *out = parsed;
        return true;
    }
    fprintf(stderr, "%s expects a positive whole number of seconds, got \"%s\"\n",
            flag, value);
    return false;
}

} // namespace

int main(int argc, char **argv) {
    std::string listen = kDefaultListen;
    std::string root = kDefaultRoot;
    HlsOptions opts;
    bool verbose = false;

    for (int i = 1; i < argc; i++) {
        const char *value = nullptr;
        if (0 == strcmp(argv[i], "--listen")) {
            if (!value_of(argc, argv, i, &value)) { return 2; }
            listen = value;
            i++;
        } else if (0 == strcmp(argv[i], "--root")) {
            if (!value_of(argc, argv, i, &value)) { return 2; }
            root = value;
            i++;
        } else if (0 == strcmp(argv[i], "--segment")) {
            if (!value_of(argc, argv, i, &value)) { return 2; }
            if (!positive_seconds("--segment", value, &opts.segment_seconds)) { return 2; }
            i++;
        } else if (0 == strcmp(argv[i], "--retention")) {
            if (!value_of(argc, argv, i, &value)) { return 2; }
            if (!positive_seconds("--retention", value, &opts.retention_seconds)) { return 2; }
            i++;
        } else if (0 == strcmp(argv[i], "--keep-all")) {
            opts.keep_all = true;
        } else if (0 == strcmp(argv[i], "--verbose")) {
            verbose = true;
        } else if (0 == strcmp(argv[i], "--help") || 0 == strcmp(argv[i], "-h")) {
            usage(stdout, argv[0]);
            return 0;
        } else {
            fprintf(stderr, "unexpected argument %s\n", argv[i]);
            usage(stderr, argv[0]);
            return 2;
        }
    }

    av_log_set_level(verbose ? AV_LOG_VERBOSE : AV_LOG_WARNING);
    avformat_network_init();

    MediaService service(root, opts);

    int real_port{0};
    grpc::ServerBuilder builder;

    // No thread cap. There used to be SetMaxThreads(2), which held the whole
    // hub to about one camera: MediaUpload occupies its thread for the life of
    // the stream, because libavformat pulls the packets from it rather than
    // being fed by one. gRPC's default pool grows with demand, and the bound on
    // concurrent cameras belongs wherever the streams are authorised, not in a
    // number chosen here.
    builder.AddListeningPort(listen, grpc::InsecureServerCredentials(), &real_port);
    builder.SetDefaultCompressionAlgorithm(GRPC_COMPRESS_NONE);
    builder.SetDefaultCompressionLevel(GRPC_COMPRESS_LEVEL_NONE);
    builder.RegisterService(&service);

    auto server = builder.BuildAndStart();
    if (server == nullptr) {
        // BuildAndStart reports a port it could not take by leaving real_port
        // at zero and returning nothing.
        fprintf(stderr, "cams: cannot listen on %s\n", listen.c_str());
        return 1;
    }
    fprintf(stderr, "cams: rtp2hls listening on %s port %d, writing under %s\n",
            listen.c_str(), real_port, root.c_str());

    server->Wait();
    avformat_network_deinit();
    return 0;
}
