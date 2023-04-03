//
// Created by jfs on 16/01/23.
//

#include <sys/mman.h>
#include <sys/stat.h>
#include <fcntl.h>

#include <iostream>
#include <cassert>

#include <archive.h>
#include <archive_entry.h>

#include "StreamStorage.hpp"
#include "MediaSource.hpp"
#include "MediaEncoder.hpp"
#include "MediaDecoder.hpp"

class ArchiveSource : public MediaSource {
public:
    ~ArchiveSource() override = default;
    ArchiveSource() = delete;
    ArchiveSource(struct archive *a) : archive_handle_{a} {}

    int Read(uint8_t *buf, size_t buf_size) override {
        struct archive_entry *entry{nullptr};
        std::cerr << __func__ << " " << __FILE__ << " +" << __LINE__<< std::endl;
        for (;;) {
            int rc = archive_read_next_header(archive_handle_, &entry);
            switch (rc) {
                case ARCHIVE_EOF:
                    std::cerr << " at " << __LINE__<< " EOF" << std::endl;
                    return AVERROR_EOF;
                case ARCHIVE_OK:
                    {
                        size_t sz = archive_entry_size(entry);
                        std::cerr << " at " << __LINE__<< " read " << sz << " into " << buf_size << std::endl;

                        assert(sz < buf_size);
                        std::string entry_path(archive_entry_pathname(entry));
                        if (!entry_path.ends_with(".rtp")) {
                            continue;
                        }
                        return archive_read_data(archive_handle_, buf, buf_size);
                    }
                default:
                    std::cerr << " at " << __LINE__<< " ERR" << std::endl;
                    return AVERROR_EXIT;
            }
        }
    }

private:
    struct archive *archive_handle_ = nullptr;
};

#define CHECK(Expectation, Value) if (Expectation != Value) { \
    auto e = archive_errno(archive_handle); \
    std::cerr << "Expected=" << Expectation << " Got=" << Value << " errno=" << e << " " << ::strerror(e) << std::endl; \
    assert(Expectation == Value); \
}

int main(int argc, char **argv) {
    static std::array<char, 16384> data_buf;
    static std::array<char, 4096> sdp_buf;

    if (2 != argc) {
        std::cerr << "Expected PATH to the input archive\n";
        return 1;
    }

    const char *archive_path = argv[1];
    std::string sdp;
    int rc;

    std::cerr << "Replaying archive [" << archive_path << "]" << std::endl;

    {
        // 1. Open the archive
        auto *archive_handle = archive_read_new();
        assert(archive_handle != NULL);
        rc = archive_read_support_filter_none(archive_handle);
        CHECK(ARCHIVE_OK, rc);
        rc = archive_read_support_format_tar(archive_handle);
        CHECK(ARCHIVE_OK, rc);
        rc = archive_read_open_filename(archive_handle, archive_path, 10240);
        CHECK(ARCHIVE_OK, rc);

        // 2. Find the SDP
        struct archive_entry *entry{nullptr};
        while (archive_read_next_header(archive_handle, &entry) == ARCHIVE_OK) {
            std::string entry_path(archive_entry_pathname(entry));
            if (entry_path.ends_with(".sdp")) {
                assert(archive_entry_size(entry) <= sdp_buf.size());
                archive_read_data(archive_handle, sdp_buf.data(), sdp_buf.size());
                sdp.assign(sdp_buf.data(), archive_entry_size(entry));
                break;
            }
        }
        assert(!sdp.empty());
        std::cerr << "SDP [" << sdp << "]" << std::endl;

        archive_read_free(archive_handle);
    }

    {
        // 3. Open the archive to iterate on the data files
        auto *archive_handle = archive_read_new();
        rc = archive_read_support_filter_none(archive_handle);
        CHECK(ARCHIVE_OK, rc);
        rc = archive_read_support_format_tar(archive_handle);
        CHECK(ARCHIVE_OK, rc);
        rc = archive_read_open_filename(archive_handle, archive_path, 10240);
        CHECK(ARCHIVE_OK, rc);

        StreamStorage storage("user", "camera");
        MediaEncoder encoder(storage);
        ArchiveSource source(archive_handle);
        MediaDecoder decoder(sdp, source, encoder);

        archive_read_free(archive_handle);
    }

    return 0;
}