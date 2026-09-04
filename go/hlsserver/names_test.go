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

import (
	"strings"
	"testing"
)

// This is the whole security boundary: the draft has no credential in front of
// it, so a name that gets through here is a file that leaves the host.
func TestSafeStreamID_Accepted(t *testing.T) {
	for _, id := range []string{
		"someone",
		// What a camera's ONVIF UUID looks like.
		"3fa85f64-5717-4562-b3fc-2c963f66afa6",
		"cam.1",
		"cam_1",
		"a",
		strings.Repeat("x", maxStreamIDLen),
	} {
		if !safeStreamID(id) {
			t.Fatalf("safeStreamID(%q) = false, want true", id)
		}
	}
}

func TestSafeStreamID_Refused(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   string
	}{
		{"empty", ""},
		{"too long", strings.Repeat("x", maxStreamIDLen+1)},
		{"the current directory", "."},
		{"the parent directory", ".."},
		{"a traversal", "../etc"},
		{"a separator", "a/b"},
		{"a backslash", "a\\b"},
		{"absolute", "/etc/passwd"},
		{"a NUL byte", "a\x00b"},
		{"a newline", "a\nb"},
		{"a space", "a b"},
		{"a percent", "a%2e%2e"},
		{"a colon", "c:"},
		{"a tilde", "~root"},
		{"non-ASCII", "café"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if safeStreamID(tc.id) {
				t.Fatalf("safeStreamID(%q) = true, want false", tc.id)
			}
		})
	}
}

func TestServableName_Accepted(t *testing.T) {
	for _, tc := range []struct {
		name string
		want servable
	}{
		{"stream.m3u8", servable{contentType: typePlaylist, live: true}},
		{"init.mp4", servable{contentType: typeInit}},
		{"seg_00000.m4s", servable{contentType: typeFragment}},
		// The muxer widens the field once it runs out of five digits.
		{"seg_1234567.m4s", servable{contentType: typeFragment}},
		{"seg_0.m4s", servable{contentType: typeFragment}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := servableName(tc.name)
			if !ok {
				t.Fatalf("servableName(%q) = false, want true", tc.name)
			}
			if got != tc.want {
				t.Fatalf("servableName(%q) = %+v, want %+v", tc.name, got, tc.want)
			}
		})
	}
}

func TestServableName_Refused(t *testing.T) {
	for _, tc := range []struct {
		name string
		file string
	}{
		// Ours, not media: it says what the directory is recording.
		{"the fingerprint", "stream.params"},
		// What the muxer renames from. Serving one would hand over a fragment
		// that is still being written.
		{"a temporary fragment", "seg_00000.m4s.tmp"},
		{"a temporary playlist", "stream.m3u8.tmp"},
		{"a traversal", "../init.mp4"},
		{"a nested path", "sub/init.mp4"},
		{"an empty fragment number", "seg_.m4s"},
		{"a non-numeric fragment", "seg_abc.m4s"},
		{"a fragment number with a sign", "seg_-1.m4s"},
		{"the wrong extension", "seg_00000.ts"},
		{"the playlist of another name", "index.m3u8"},
		{"empty", ""},
		{"the current directory", "."},
		{"the parent directory", ".."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, ok := servableName(tc.file); ok {
				t.Fatalf("servableName(%q) = true, want false", tc.file)
			}
		})
	}
}

// The two halves have to agree, because one names the directories and the other
// reads them. This is the shape of the check on the C++ side.
func TestSafeStreamID_MatchesTheCppRule(t *testing.T) {
	// Every byte the C++ rule accepts, and nothing else.
	for c := 0; c < 256; c++ {
		id := string([]byte{byte(c)})
		want := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '.' || c == '-' || c == '_'
		// "." is accepted as a character but refused as a whole identifier.
		if id == "." {
			want = false
		}
		if got := safeStreamID(id); got != want {
			t.Fatalf("safeStreamID(%q) = %v, want %v (byte %d)", id, got, want, c)
		}
	}
}
