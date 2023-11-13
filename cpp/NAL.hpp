//
// Created by jfs on 15/05/23.
//

#pragma once

#include <cassert>
#include <cstdint>
#include <list>

enum class NALUType : uint8_t {
    NonIDR = 1,
    DataPartitionA = 2,
    DataPartitionB = 3,
    DataPartitionC = 4,
    IDR = 5,
    SEI = 6,
    SPS = 7,
    PPS = 8,
    AccessUnitDelimiter = 9,
    EndOfSequence = 10,
    EndOfStream = 11,
    FillerData = 12,
    SPSExtension = 13,
    Prefix = 14,
    SubsetSPS = 15,
    Reserved16 = 16,
    Reserved17 = 17,
    Reserved18 = 18,
    SliceLayerWithoutPartitioning = 19,
    SliceExtension = 20,
    SliceExtensionDepth = 21,
    Reserved22 = 22,
    Reserved23 = 23,

    // additional NALU types for RTP/H264
    STAPA = 24,
    STAPB = 25,
    MTAP16 = 26,
    MTAP24 = 27,
    FUA = 28,
    FUB = 29,
};

struct NALUHeader {
    uint8_t forbidden: 1 = 0u;
    uint8_t idc: 2 = 0u;
    uint8_t type: 5 = 0u;

    void parse(const uint8_t *buf, std::size_t len) {
        assert(len >= sizeof(*this));
        forbidden = (buf[0] >> 7) & 0x01;
        idc = (buf[0] >> 5) & 0x03;
        type = buf[0] & 0x1F;
    }
};

static_assert(sizeof(NALUHeader) == 1);

struct NALU {
    NALUHeader header;
    const uint8_t *buf = nullptr;
    std::size_t length = 0;

    NALU(const uint8_t *b, std::size_t sz) : buf{b}, length{sz} {
        header.parse(b, sz);
    }
};

std::list<NALU> parse_train(const uint8_t *buf, const std::size_t len) {
    enum nal_parsing_state {DELIM_1ST = 0, DELIM_2ND, DELIM_3RD, DELIM_ONE};
    nal_parsing_state state = DELIM_1ST;

    std::list<NALU> nalus;
    const uint8_t *start = buf;
    const uint8_t *p = buf;
    for (std::size_t l=0; l<len; l++, p++) {
        switch (state) {
            case DELIM_1ST:
                state = (*p == 0x00) ? DELIM_2ND : DELIM_1ST;
                continue;
            case DELIM_2ND:
                state = (*p == 0x00) ? DELIM_3RD : DELIM_1ST;
                continue;
            case DELIM_3RD:
                state = (*p == 0x00) ? DELIM_ONE : DELIM_1ST;
                continue;
            case DELIM_ONE:
                if (*p == 0x01) {
                    // end of a black marker! the last block ended 4 bytes earlier
                    const uint8_t *end = p - 4;
                    if (start != end)
                        nalus.emplace_back(start, end - start);
                    start = p+1;
                    state = DELIM_1ST;
                }
                continue;
        }
    }

    if (start < p) {
        nalus.emplace_back(start, p - start);
    }
    return nalus;
}