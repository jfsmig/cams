//
// Created by jfs on 12/05/23.
//

#pragma once

#include <cstdint>
#include <cstdio>

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

static inline void hex(const uint8_t* buf, const size_t len, const size_t MAX = 64);

void hex(const uint8_t* buf, const size_t len, const size_t MAX) {
    for (size_t i=0; i < len && i < MAX;) {
        for (size_t k=0; i < len && k<4; k++, i++) {
            ::fprintf(stderr, "%02x", buf[i]);
        }
        ::fputs( " ", stderr);
        for (size_t k=0; i < len && k<4; k++, i++) {
            ::fprintf(stderr, "%02x", buf[i]);
        }
        ::fputs("\n", stderr);
    }
}