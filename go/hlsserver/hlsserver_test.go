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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testUser = "someone"
	testCam  = "3fa85f64-5717-4562-b3fc-2c963f66afa6"
)

// newFixture lays out a root the way cams-rtp2hls would, including the two
// files that must never be served.
func newFixture(t *testing.T) (*Server, string) {
	t.Helper()

	root := t.TempDir()
	dir := filepath.Join(root, testUser, testCam)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", name, err)
		}
	}
	write(namePlaylist, "#EXTM3U\n#EXT-X-MAP:URI=\"init.mp4\"\nseg_00000.m4s\n")
	write(nameInit, "init-bytes")
	write("seg_00000.m4s", "0123456789")
	// Ours, not media.
	write("stream.params", "h264 640x480 22:aaaa")
	// What the muxer renames from.
	write("seg_00001.m4s.tmp", "half-written")

	srv, err := New(Config{Root: root, ListenAddr: "127.0.0.1:0"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return srv, root
}

func get(t *testing.T, srv *Server, method, target string, header http.Header) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	for k, v := range header {
		req.Header[k] = v
	}
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

func mediaURL(file string) string {
	return "/hls/" + testUser + "/" + testCam + "/" + file
}

func TestNew_RefusesAnEmptyConfig(t *testing.T) {
	if _, err := New(Config{ListenAddr: "127.0.0.1:0"}); err == nil {
		t.Fatal("an empty root was accepted")
	}
	if _, err := New(Config{Root: "/tmp"}); err == nil {
		t.Fatal("an empty listen address was accepted")
	}
}

func TestServeMedia_Playlist(t *testing.T) {
	srv, _ := newFixture(t)

	rec := get(t, srv, http.MethodGet, mediaURL(namePlaylist), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != typePlaylist {
		t.Fatalf("content type %q, want %q", got, typePlaylist)
	}
	// Rewritten in place on every fragment, so a cached copy is a stalled
	// player.
	if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("cache control %q, want no-cache", got)
	}
	if !strings.HasPrefix(rec.Body.String(), "#EXTM3U") {
		t.Fatalf("body does not look like a playlist: %q", rec.Body.String())
	}
}

func TestServeMedia_FragmentIsCachedButNotImmutable(t *testing.T) {
	srv, _ := newFixture(t)

	rec := get(t, srv, http.MethodGet, mediaURL("seg_00000.m4s"), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != typeFragment {
		t.Fatalf("content type %q, want %q", got, typeFragment)
	}
	cache := rec.Header().Get("Cache-Control")
	if !strings.Contains(cache, "max-age=") {
		t.Fatalf("cache control %q carries no max-age", cache)
	}
	// A Fresh reconcile restarts fragment numbering at zero, so this name comes
	// to hold different bytes. "immutable" would be a lie with a year on it.
	if strings.Contains(cache, "immutable") {
		t.Fatalf("cache control %q says immutable, which a fragment is not", cache)
	}
}

func TestServeMedia_Range(t *testing.T) {
	srv, _ := newFixture(t)

	rec := get(t, srv, http.MethodGet, mediaURL("seg_00000.m4s"),
		http.Header{"Range": {"bytes=2-5"}})
	if rec.Code != http.StatusPartialContent {
		t.Fatalf("status %d, want 206", rec.Code)
	}
	if got := rec.Body.String(); got != "2345" {
		t.Fatalf("body %q, want %q", got, "2345")
	}
}

func TestServeMedia_Head(t *testing.T) {
	srv, _ := newFixture(t)

	rec := get(t, srv, http.MethodHead, mediaURL("seg_00000.m4s"), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Content-Length"); got != "10" {
		t.Fatalf("content length %q, want 10", got)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("HEAD returned %d bytes of body", rec.Body.Len())
	}
}

// The allowlist is the reason these are unreachable, and none of them has to be
// named to make it so.
func TestServeMedia_Refused(t *testing.T) {
	srv, _ := newFixture(t)

	for _, tc := range []struct {
		name   string
		target string
	}{
		{"the fingerprint", mediaURL("stream.params")},
		{"a temporary fragment", mediaURL("seg_00001.m4s.tmp")},
		{"an unknown file", mediaURL("secrets.txt")},
		{"a fragment that does not exist", mediaURL("seg_09999.m4s")},
		{"an unknown camera", "/hls/" + testUser + "/nope/" + namePlaylist},
		{"an unknown user", "/hls/nobody/" + testCam + "/" + namePlaylist},
		{"a traversal in the camera", "/hls/" + testUser + "/../../etc/" + namePlaylist},
		{"a traversal in the file", mediaURL("..%2Finit.mp4")},
		{"a dot as the camera", "/hls/" + testUser + "/./" + namePlaylist},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := get(t, srv, http.MethodGet, tc.target, nil)
			switch rec.Code {
			case http.StatusNotFound:
				// One answer for every reason: anything more would be a
				// directory oracle.
				if body := rec.Body.String(); !strings.Contains(body, "not found") {
					t.Fatalf("body %q says more than \"not found\"", body)
				}
			case http.StatusMovedPermanently, http.StatusTemporaryRedirect,
				http.StatusPermanentRedirect:
				// ServeMux normalises a path holding "." or ".." and redirects
				// to the cleaned form before any handler runs. The cleaned form
				// matches no route here, so the follow-up is a 404 and nothing
				// is served either way -- which is what
				// TestServeMedia_TraversalLeaksNothing asserts directly. Not
				// fought, because defeating it would mean a middleware in front
				// of the mux for no gain in what is reachable.
				if to := rec.Header().Get("Location"); strings.Contains(to, "..") {
					t.Fatalf("redirected to an uncleaned path %q", to)
				}
			default:
				t.Fatalf("status %d for %s, want 404 or a redirect to a cleaned path",
					rec.Code, tc.target)
			}
		})
	}
}

// Whatever the path did, the answer must not carry a byte of the file the
// traversal was aiming at.
func TestServeMedia_TraversalLeaksNothing(t *testing.T) {
	srv, root := newFixture(t)

	secret := filepath.Join(root, "secret.txt")
	if err := os.WriteFile(secret, []byte("SENTINEL"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	for _, target := range []string{
		"/hls/" + testUser + "/" + testCam + "/../../secret.txt",
		"/hls/../../secret.txt/x/" + namePlaylist,
		"/hls/" + testUser + "/" + testCam + "/%2e%2e%2f%2e%2e%2fsecret.txt",
	} {
		rec := get(t, srv, http.MethodGet, target, nil)
		if strings.Contains(rec.Body.String(), "SENTINEL") {
			t.Fatalf("%s returned the file it was aiming at", target)
		}
		if rec.Code == http.StatusOK {
			t.Fatalf("%s was answered with 200", target)
		}
	}
}

func TestServeStreams(t *testing.T) {
	srv, root := newFixture(t)

	// A camera that has a directory and has never streamed, which is what a
	// registered camera nobody asked to play looks like.
	if err := os.MkdirAll(filepath.Join(root, testUser, "idle-cam"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	rec := get(t, srv, http.MethodGet, "/streams", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", rec.Code)
	}

	var got []streamEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding %q: %v", rec.Body.String(), err)
	}
	if len(got) != 2 {
		t.Fatalf("listed %d streams, want 2: %+v", len(got), got)
	}
	// Sorted, so two calls agree and the picker does not reshuffle.
	if got[0].Camera != testCam || !got[0].Playlist {
		t.Fatalf("first entry %+v, want the streaming camera", got[0])
	}
	if got[1].Camera != "idle-cam" || got[1].Playlist {
		t.Fatalf("second entry %+v, want the idle camera", got[1])
	}
}

func TestServeStreams_EmptyRoot(t *testing.T) {
	srv, err := New(Config{Root: t.TempDir(), ListenAddr: "127.0.0.1:0"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rec := get(t, srv, http.MethodGet, "/streams", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", rec.Code)
	}
	// An empty list, not null: the page iterates it.
	if got := strings.TrimSpace(rec.Body.String()); got != "[]" {
		t.Fatalf("body %q, want []", got)
	}
}

func TestServePlayer(t *testing.T) {
	srv, _ := newFixture(t)

	for _, target := range []string{"/", "/watch/" + testUser + "/" + testCam} {
		rec := get(t, srv, http.MethodGet, target, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d for %s, want 200", rec.Code, target)
		}
		if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/html") {
			t.Fatalf("content type %q for %s", got, target)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "<video") {
			t.Fatalf("%s served a page with no video element", target)
		}
		// The CDN script has to stay pinned and checked.
		if !strings.Contains(body, "integrity=\"sha384-") {
			t.Fatalf("%s served hls.js with no integrity hash", target)
		}
	}
}

// A bad identifier is one answer everywhere, not something the page discovers.
func TestServePlayer_RefusesABadIdentifier(t *testing.T) {
	srv, _ := newFixture(t)

	rec := get(t, srv, http.MethodGet, "/watch/"+testUser+"/../etc", nil)
	if rec.Code == http.StatusOK {
		t.Fatal("a traversal in a watch URL was answered with the player")
	}
}

func TestServeHealth(t *testing.T) {
	srv, _ := newFixture(t)

	rec := get(t, srv, http.MethodGet, "/healthz", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != "ok" {
		t.Fatalf("body %q, want ok", got)
	}
}

// Only GET and HEAD are answered; anything else is the mux's business but the
// result has to be a refusal.
func TestServeMedia_RefusesOtherMethods(t *testing.T) {
	srv, _ := newFixture(t)

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		rec := get(t, srv, method, mediaURL(namePlaylist), nil)
		if rec.Code == http.StatusOK {
			t.Fatalf("%s was answered with 200", method)
		}
	}
}
