//
// Created by jfs on 12/05/23.
//

#pragma once

#include <arpa/inet.h>

#include <cstdint>
#include <cstring>
#include <cassert>

#include "bits.hpp"

struct RtpHeader {
    uint8_t version: 2 = 0U;
    uint8_t padding: 1 = 0U;
    uint8_t extension: 1 = 0U;
    uint8_t crsc_count: 4 = 0U;

    uint8_t marker: 1 = 0U;
    uint8_t payload_type: 7 = 0U;

    uint16_t sequence_number = 0U;
    uint32_t timestamp = 0U;
    uint32_t ssrc_id = 0U;

    std::size_t parse(const uint8_t *buf, std::size_t len) {
        assert(len > sizeof(*this));

        // byte 0
        version = (buf[0] >> 6) & 0x03;
        padding = (buf[0] >> 5) & 0x01;
        extension = (buf[0] >> 4) & 0x01;
        crsc_count = bitflip4(buf[0] & 0x0F);
        // byte 1
        marker = (buf[1] >> 7) & 0x01;
        payload_type = buf[1] & 0x7F;
        // other bytes
        sequence_number = ntohs(*reinterpret_cast<const uint16_t *>(buf + 2)); // 2,3
        timestamp = ntohs(*reinterpret_cast<const uint32_t *>(buf + 4)); // 4,5,6,7
        ssrc_id = ntohs(*reinterpret_cast<const uint32_t *>(buf + 8)); // 8,9,10,11
        return sizeof(*this);
    }
} __attribute__((packed));

struct RtpHeaderExtension {
    struct Preamble { ;
        uint16_t id = 0;
        uint16_t length = 0;
    } preamble;
    const uint8_t *data = nullptr;

    std::size_t parse(const uint8_t *buf, std::size_t len) {
        assert(len >= sizeof(Preamble));

        preamble.id = *reinterpret_cast<const uint16_t *>(buf);
        preamble.length = ntohs(*reinterpret_cast<const uint16_t *>(buf + 2));
        if (preamble.length > 0) {
            data = buf + sizeof(preamble);
        }
        return sizeof(preamble) + preamble.length;
    }
} __attribute__((packed));

static_assert(12 == sizeof(RtpHeader));
static_assert(4 == sizeof(RtpHeaderExtension::Preamble));

struct RtpPacket {
    RtpHeader header;
    RtpHeaderExtension extension;
    const uint8_t *payload = nullptr;
    std::size_t payload_size = 0;
    uint32_t csrc_id[16] = {0};

    static RtpPacket parse(const uint8_t *buf, std::size_t len) {
        RtpPacket pkt;
        std::size_t consumed;

        consumed = pkt.header.parse(buf, len);
        buf += consumed, len -= consumed;

        for (int i = 0; i < pkt.header.crsc_count; i++) {
            memcpy(pkt.csrc_id + i, buf, sizeof(uint32_t));
            consumed = sizeof(uint32_t);
            buf += consumed, len -= consumed;
        }

        if (pkt.header.extension) {
            consumed = pkt.extension.parse(buf, len);
            buf += consumed, len -= consumed;
        }

        if (len > 0) {
            pkt.payload = buf;
            pkt.payload_size = len;
        }
        return pkt;
    }
};
