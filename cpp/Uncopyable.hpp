//
// Created by jfs on 14/01/23.
//

#ifndef CAMS_CPP_UNCOPYABLE_HPP
#define CAMS_CPP_UNCOPYABLE_HPP


class Uncopyable {
public:
    Uncopyable() = default;

    Uncopyable(Uncopyable &&rhs) = delete;

    Uncopyable(const Uncopyable &rhs) = delete;
};


#endif //CAMS_CPP_UNCOPYABLE_HPP
