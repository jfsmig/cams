//
// Created by jfs on 12/05/23.
//

#pragma once

#include <arpa/inet.h>

#include <cstdint>

// Network/Host bits order flip for 4 bits integers.
static inline uint8_t bitflip1(uint8_t x) __attribute__((pure, hot));
static inline uint8_t bitflip2(uint8_t x) __attribute__((pure, hot));
static inline uint8_t bitflip3(uint8_t x) __attribute__((pure, hot));
static inline uint8_t bitflip4(uint8_t x) __attribute__((pure, hot));
static inline uint8_t bitflip5(uint8_t x) __attribute__((pure, hot));
static inline uint8_t bitflip6(uint8_t x) __attribute__((pure, hot));
static inline uint8_t bitflip7(uint8_t x) __attribute__((pure, hot));
static inline uint8_t bitflip8(uint8_t x) __attribute__((pure, hot));

// 0 ops
uint8_t bitflip1(uint8_t x) { return x; }
// 2 ops
uint8_t bitflip2(uint8_t x) { return bitflip4(x) >> 2; }
// 2 ops
uint8_t bitflip3(uint8_t x) { return bitflip4(x) >> 1; }
// 1 ops
uint8_t bitflip4(uint8_t x) {
    static const uint8_t map4[16] = {
            {0x00},{0x08},{0x04},{0x0C},
            {0x02},{0x0A},{0x06},{0x0E},
            {0x01},{0x09},{0x05},{0x0D},
            {0x03},{0x0B},{0x07},{0x0F},
    };
    return map4[x];
}
// 5 ops
uint8_t bitflip6(uint8_t x) { return bitflip8(x) >> 2; }
// 5 ops
uint8_t bitflip5(uint8_t x) { return bitflip8(x) >> 3; }
// 5 ops
uint8_t bitflip7(uint8_t x) { return bitflip8(x) >> 1; }
// 4 ops
uint8_t bitflip8(uint8_t x) { return (bitflip4(x & 0x0F) << 4) | bitflip4((x & 0xF0) >> 4); }

struct RtpHeaderCore {
    uint8_t version : 2;
    uint8_t padding : 1;
    uint8_t extension : 1;
    uint8_t crsc_count : 4;

    uint8_t marker : 1;
    uint8_t payload_type : 7;

    uint16_t sequence_number;
    uint32_t timestamp;
    uint32_t ssrc_id;  // synchronisation sourceID

    void ntoh() {
        crsc_count = bitflip4(crsc_count);
        sequence_number = ntohs(sequence_number);
        timestamp = ntohl(timestamp);
        ssrc_id = ntohl(ssrc_id);
    }

    void hton() {
        crsc_count = bitflip4(crsc_count);
        sequence_number = htonl(sequence_number);
        timestamp = htonl(timestamp);
        ssrc_id = htonl(ssrc_id);
    }
} __attribute__((packed));


struct RtpHeaderExtension {
    uint16_t id;
    uint16_t length;
    uint8_t header;

    void ntoh() {
        length = ntohs(length);
    }

    void hton() {
        length = htons(length);
    }

} __attribute__((packed));


template <unsigned int NB_CONTRIB>
struct RtpHeader {
    RtpHeaderCore header;
    uint32_t csrc_id[NB_CONTRIB];
    RtpHeaderExtension ext;

    void ntoh() {
        header.ntoh();
        ext.ntoh();
        for (auto &id : csrc_id) {
            id = ntohl(id);
        }
    }

    void hton() {
        header.hton();
        ext.hton();
        for (auto &id : csrc_id) {
            id = hton(id);
        }
    }

    static RtpHeader<NB_CONTRIB> parse(const uint8_t *buf) {
        RtpHeader hdr = *reinterpret_cast<const RtpHeader<NB_CONTRIB>*>(buf);
        hdr.header.ntoh();
        return hdr;
    }
} __attribute__((packed));


template <unsigned int NB_CONTRIB>
struct RtpPacket {
    RtpHeader<NB_CONTRIB> header;
    uint8_t payload[];

    void ntoh() {
        header.ntoh();
    }

    void hton() {
        header.hton();
    }

} __attribute__((packed));
