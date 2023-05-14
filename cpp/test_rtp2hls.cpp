//
// Created by jfs on 16/01/23.
//

#include <sys/mman.h>

#include <iostream>
#include <cassert>

#include <archive.h>
#include <archive_entry.h>

#include "StreamStorage.hpp"
#include "MediaEncoder.hpp"
#include "MediaDecoder.hpp"

#define CHECK(Expectation, Value) if (Expectation != Value) { \
    auto e = archive_errno(archive_handle); \
    std::cerr << "Expected=" << Expectation << " Got=" << Value << " errno=" << e << " " << ::strerror(e) << std::endl; \
    assert(Expectation == Value); \
}

int main(int argc, char **argv) {
    static std::array<char, 32768> data_buf;

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
                assert(archive_entry_size(entry) <= data_buf.size());
                archive_read_data(archive_handle, data_buf.data(), data_buf.size());
                sdp.assign(data_buf.data(), archive_entry_size(entry));
                break;
            }
        }
        assert(!sdp.empty());
        archive_read_free(archive_handle);
    }
    std::cerr << "SDP [" << sdp << "]" << std::endl;

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
        MediaDecoder decoder(sdp, encoder);

        // Only trigger the decoder for the RTP files
        struct archive_entry *entry{nullptr};
        int i=0;
        while (archive_read_next_header(archive_handle, &entry) == ARCHIVE_OK) {
            if (i++ > 10) break;
            std::string entry_path (archive_entry_pathname (entry));
            if (!entry_path.ends_with (".rtp"))
                continue;
            const size_t actual_size = archive_entry_size (entry);
            std::cerr << "RTP " << entry_path << " size=" << actual_size << std::endl;
            assert (actual_size <= data_buf.size ());
            auto r = archive_read_data (archive_handle, data_buf.data (), data_buf.size ());
            assert(r == actual_size);
            decoder.on_rtp(reinterpret_cast<uint8_t*>(data_buf.data()), actual_size);
        }

        encoder.flush();
        archive_read_free(archive_handle);
    }

    return 0;
}