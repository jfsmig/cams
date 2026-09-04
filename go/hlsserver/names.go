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

package hlsserver

import "strings"

// maxStreamIDLen bounds an identifier, as the C++ side does.
const maxStreamIDLen = 128

// The media types this server hands out. Set explicitly because Go's mime
// package does not reliably know the last two, and because Safari refuses a
// playlist served as anything but the first.
const (
	typePlaylist = "application/vnd.apple.mpegurl"
	typeInit     = "video/mp4"
	typeFragment = "video/iso.segment"
)

// The names the muxer writes. Kept here rather than derived, because they are a
// contract with StreamStorage on the C++ side and not a local choice.
const (
	namePlaylist   = "stream.m3u8"
	nameInit       = "init.mp4"
	fragmentPrefix = "seg_"
	fragmentSuffix = ".m4s"
)

// safeStreamID reports whether an identifier may be used as a path component.
//
// The same rule as stream_id_is_safe in cpp/hub/StreamStorage.cpp, and it has
// to stay the same: one side names those directories and this one reads them.
// Non-empty, short enough, and made only of characters that cannot escape a
// directory -- neither '/' nor '\' nor NUL is in the set, which is what makes
// traversal impossible here rather than merely awkward.
//
// Compared byte by byte, so any non-ASCII input is refused rather than
// normalised. A camera's ONVIF UUID is ASCII; anything else did not come from
// one.
func safeStreamID(id string) bool {
	if id == "" || len(id) > maxStreamIDLen {
		return false
	}
	if id == "." || id == ".." {
		return false
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		ok := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '.' || c == '-' || c == '_'
		if !ok {
			return false
		}
	}
	return true
}

// servable describes a file this server will hand over.
type servable struct {
	contentType string

	// live marks the playlist, which the muxer rewrites in place as the stream
	// grows. Everything else is written once under a temporary name and
	// renamed, so it can be cached for a while; this cannot be cached at all.
	live bool
}

// servableName reports whether a file in a stream's directory may be served.
//
// An allowlist rather than a filter. The directory also holds stream.params --
// our own fingerprint, not media -- and the ".tmp" files the muxer renames
// from, and neither has any business leaving the host. Naming what may be
// served keeps both unreachable without having to enumerate them, and keeps a
// file added later unreachable by default.
func servableName(name string) (servable, bool) {
	switch name {
	case namePlaylist:
		return servable{contentType: typePlaylist, live: true}, true
	case nameInit:
		return servable{contentType: typeInit}, true
	}

	if !strings.HasPrefix(name, fragmentPrefix) ||
		!strings.HasSuffix(name, fragmentSuffix) {
		return servable{}, false
	}
	digits := name[len(fragmentPrefix) : len(name)-len(fragmentSuffix)]
	if digits == "" {
		return servable{}, false
	}
	for i := 0; i < len(digits); i++ {
		if digits[i] < '0' || digits[i] > '9' {
			return servable{}, false
		}
	}
	return servable{contentType: typeFragment}, true
}
