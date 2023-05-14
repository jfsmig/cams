//
// Created by jfs on 15/05/23.
//

#pragma once

#include <cstdint>

enum class NALUType : uint8_t {
    NonIDR = 1;
    DataPartitionA = 2;
    DataPartitionB = 3;
    DataPartitionC = 4;
    IDR = 5;
    SEI = 6;
    SPS = 7;
    PPS = 8;
    AccessUnitDelimiter = 9;
    EndOfSequence = 10;
    EndOfStream = 11;
    FillerData = 12;
    SPSExtension = 13;
    Prefix = 14;
    SubsetSPS = 15;
    Reserved16 = 16;
    Reserved17 = 17;
    Reserved18 = 18;
    SliceLayerWithoutPartitioning = 19;
    SliceExtension = 20;
    SliceExtensionDepth = 21;
    Reserved22 = 22;
    Reserved23 = 23;

    // additional NALU types for RTP/H264
    STAPA = 24;
    STAPB = 25;
    MTAP16 = 26;
    MTAP24 = 27;
    FUA = 28;
    FUB = 29;
};

struct NALUHeader {
    uint8_t forbidden: 1 = 0u;
    uint8_t idc: 2 = 0u;
    uint8_t type: 5 = 0u;

    std::size_t parse(const uint8_t *buf, std::size_t len) {
        if (len < 1) return 0;
        forbidden = (buf[0] >> 7) & 0x01;
        idc = (buf[0] >> 5) & 0x03;
        type = buf[0] & 0x1F;
        return sizeof(*this);
    }
};

struct NALU {
    NALUHeader header;
    const uint8_t *buf = nullptr;
    size_t length = 0;
};

static_assert(sizeof(NALUHeader) == 1);