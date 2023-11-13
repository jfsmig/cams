//
// Created by jfs on 12/05/23.
//

#include <iostream>
#include <cassert>

#include "bits.hpp"

#define run(Max,Func) do {                      \
    for (int i=0; i<Max; i++) {                 \
        uint8_t x = static_cast<uint8_t>(i);    \
        /*fprintf(stderr, "%d -> %02x -> %02x\n", i, x, Func(x));*/ \
        assert(Func(Func(x)) == x);             \
    }                                           \
} while (0)

int main(int argc, char **argv) {
    (void) argc, (void) argv;
    run(4,bitflip2);
    run(8,bitflip3);
    run(16,bitflip4);
    run(32,bitflip5);
    run(64,bitflip6);
    run(128,bitflip7);
    run(256,bitflip8);
    return 0;
}