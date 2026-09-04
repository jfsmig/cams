// Copyright (c) 2022-2024 The authors (see the AUTHORS file)
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
