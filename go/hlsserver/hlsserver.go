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
// Package hlsserver serves a hub's stored fragments to a browser.
//
// It reads what cams-rtp2hls wrote and nothing else: a playlist, an
// initialisation segment and the fMP4 fragments beside them, under
// <root>/<user>/<camera>/. The layout is StreamStorage's on the C++ side, and
// this package is a reader of it -- in particular it never rewrites a playlist,
// because the muxer already writes relative URIs and the URL space mirrors the
// directory.
//
// **It has no authorization.** Anyone who can reach the port can watch every
// camera and list them, which is why the listener belongs on loopback and why
// path validation here is the only boundary there is. See names.go.
package hlsserver

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"

	"github.com/juju/errors"
)

// cacheControlMedia is deliberately short, and deliberately not "immutable".
//
// A fragment looks immutable and is not. On a Fresh reconcile -- a codec
// change, or a first run after the directory was emptied -- StreamStorage
// empties the directory and the muxer restarts fragment numbering at zero, so
// seg_00000.m4s comes to hold different bytes than it did before. A cache told
// "immutable" would serve the old recording for a year.
//
// The fix belongs on the writer's side when a CDN makes it worth having: either
// hls_start_number_source=epoch, so names never collide across restarts, or a
// generation segment in the path. Until then this is short enough that a stale
// copy costs one fragment, and the page cache is doing the real work anyway.
const cacheControlMedia = "private, max-age=30"

// Config is what a caller chooses about the server.
type Config struct {
	// Root is the directory cams-rtp2hls writes under: the same --root, on the
	// same host or the same mount. The directory is the only state the two
	// processes share.
	Root string

	// ListenAddr is where to accept viewers. Loopback unless somebody
	// deliberately says otherwise; there is no authorization.
	ListenAddr string
}

// Server is the HTTP surface over one root.
type Server struct {
	cfg Config
}

// New checks the configuration and builds the server.
func New(cfg Config) (*Server, error) {
	if cfg.Root == "" {
		return nil, errors.NotValidf("empty root")
	}
	if cfg.ListenAddr == "" {
		return nil, errors.NotValidf("empty listen address")
	}
	return &Server{cfg: cfg}, nil
}

// Handler routes every request this server answers.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.serveHealth)
	mux.HandleFunc("GET /streams", s.serveStreams)
	mux.HandleFunc("GET /hls/{user}/{camera}/{file}", s.serveMedia)
	mux.HandleFunc("GET /watch/{user}/{camera}", s.servePlayer)
	mux.HandleFunc("GET /{$}", s.servePlayer)
	return mux
}

// stream is one camera's directory.
//
// Resolved before anything is opened, rather than joining a path at the point
// of use, so that the low-latency path has somewhere to attach: serving an
// LL-HLS part means parking a request until the part exists, and a waiter needs
// an object to wait on.
type stream struct {
	user   string
	camera string
	dir    string
}

// resolve validates the identifiers and locates the directory. The identifiers
// come from a URL, which is to say from a stranger, and they become path
// components.
func (s *Server) resolve(user, camera string) (stream, bool) {
	if !safeStreamID(user) || !safeStreamID(camera) {
		return stream{}, false
	}
	return stream{
		user:   user,
		camera: camera,
		dir:    filepath.Join(s.cfg.Root, user, camera),
	}, true
}

func (st stream) path(name string) string { return filepath.Join(st.dir, name) }

// notFound is the only thing this server will say about a stream it will not
// serve.
//
// One answer for every reason -- no such user, no such camera, never streamed,
// a name that is not servable, a fragment that has fallen off the window --
// because telling them apart turns this into a directory oracle.
func notFound(w http.ResponseWriter) {
	http.Error(w, "not found", http.StatusNotFound)
}

func (s *Server) serveHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if _, err := w.Write([]byte("ok\n")); err != nil {
		// Nothing to do about a client that left mid-answer.
		return
	}
}

// serveMedia hands over one file of one stream.
func (s *Server) serveMedia(w http.ResponseWriter, r *http.Request) {
	st, ok := s.resolve(r.PathValue("user"), r.PathValue("camera"))
	if !ok {
		notFound(w)
		return
	}
	name := r.PathValue("file")
	what, ok := servableName(name)
	if !ok {
		notFound(w)
		return
	}

	// Opened before being described, so that a file renamed away underneath
	// this request is still served whole: the muxer writes under a temporary
	// name and renames, so the descriptor outlives the name.
	f, err := os.Open(st.path(name))
	if err != nil {
		notFound(w)
		return
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			return
		}
	}()

	info, err := f.Stat()
	if err != nil || info.IsDir() {
		notFound(w)
		return
	}

	w.Header().Set("Content-Type", what.contentType)
	if what.live {
		// Rewritten in place every time a fragment completes, so a cached copy
		// is a stalled player.
		w.Header().Set("Cache-Control", "no-cache")
	} else {
		w.Header().Set("Cache-Control", cacheControlMedia)
	}

	// ServeContent rather than ServeFile: it gives range requests, HEAD,
	// If-Modified-Since and an ETag without ServeFile's path cleaning and
	// redirects, which have no business anywhere near a validated path.
	http.ServeContent(w, r, name, info.ModTime(), f)
}

// streamEntry is one camera, as /streams reports it.
type streamEntry struct {
	User   string `json:"user"`
	Camera string `json:"camera"`

	// Playlist says whether the camera has ever streamed. A registered camera
	// nobody has asked to play has a directory and nothing in it.
	Playlist bool `json:"playlist"`
}

// serveStreams lists what is on disk.
//
// It exists because the identifiers are ONVIF UUIDs and nobody types those, so
// without it the player cannot be pointed at anything. It gives away no more
// than the media endpoint already does while there is no authorization -- but
// it is **the first thing that has to be gated** when a credential arrives,
// because it turns "watch a camera you were told about" into "enumerate every
// camera on the hub".
func (s *Server) serveStreams(w http.ResponseWriter, _ *http.Request) {
	out := make([]streamEntry, 0)

	users, err := os.ReadDir(s.cfg.Root)
	if err != nil {
		// An empty root and an unreadable one look the same to a viewer, which
		// is the right amount to say.
		users = nil
	}
	for _, user := range users {
		if !user.IsDir() || !safeStreamID(user.Name()) {
			continue
		}
		cameras, err := os.ReadDir(filepath.Join(s.cfg.Root, user.Name()))
		if err != nil {
			continue
		}
		for _, camera := range cameras {
			if !camera.IsDir() || !safeStreamID(camera.Name()) {
				continue
			}
			st, ok := s.resolve(user.Name(), camera.Name())
			if !ok {
				continue
			}
			_, serr := os.Stat(st.path(namePlaylist))
			out = append(out, streamEntry{
				User:     user.Name(),
				Camera:   camera.Name(),
				Playlist: serr == nil,
			})
		}
	}

	// Sorted so that two calls agree and the picker does not reshuffle.
	sort.Slice(out, func(i, j int) bool {
		if out[i].User != out[j].User {
			return out[i].User < out[j].User
		}
		return out[i].Camera < out[j].Camera
	})

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	if err := json.NewEncoder(w).Encode(out); err != nil {
		// The header is already out; there is nowhere to report this but here.
		return
	}
}

// playerHTML is the demonstration page. In this package rather than in
// frontend/, deliberately: it exists to prove the media path works in a real
// browser, and calling it the Frontend that ARCHITECTURE.md describes would
// overclaim by a wide margin.
//
//go:embed player.html
var playerHTML string

// servePlayer answers both "/" and "/watch/{user}/{camera}" with the same page,
// which reads its own URL to decide whether it has a camera already.
func (s *Server) servePlayer(w http.ResponseWriter, r *http.Request) {
	if user := r.PathValue("user"); user != "" {
		// Refused here rather than left for the page to discover, so that a
		// bad identifier is one answer everywhere.
		if _, ok := s.resolve(user, r.PathValue("camera")); !ok {
			notFound(w)
			return
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	if _, err := w.Write([]byte(playerHTML)); err != nil {
		return
	}
}
