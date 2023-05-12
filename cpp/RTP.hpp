//
// Created by jfs on 12/05/23.
//

#pragma once

#include <arpa/inet.h>

#include <cstdint>

#include "bits.hpp"

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
