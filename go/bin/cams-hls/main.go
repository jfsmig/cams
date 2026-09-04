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
// cams-hls serves the hub's stored fragments to a browser.
//
// It is the reader of what cams-rtp2hls writes, and it carries a small player
// page so that the media path can be seen working rather than only tested.
//
// **There is no authorization.** Anyone who can reach the port can watch every
// camera on the hub and list them, so the listener defaults to loopback and
// belongs behind something that does authenticate before it is exposed.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/jfsmig/cams/go/hlsserver"
	"github.com/jfsmig/cams/go/utils"
	"github.com/spf13/cobra"
)

const (
	// One port along from cams-rtp2hls, which is one along from cams-ctrl.
	defaultListenAddr = "127.0.0.1:6002"

	// The same default as cams-rtp2hls --root. The directory is the only state
	// the two processes share, so they have to agree on it.
	defaultRoot = "/var/lib/cams/hls"

	// A fragment is a short answer and a playlist a shorter one, so these are
	// generous. Serving an LL-HLS part will mean parking a request until the
	// part exists, which needs the write timeout relaxed or lifted for that one
	// route; see the plan's note on blocking playlist reload.
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 60 * time.Second
	idleTimeout       = 120 * time.Second

	// How long a shutdown waits for requests already in flight.
	shutdownGrace = 5 * time.Second
)

func main() {
	cfg := hlsserver.Config{
		Root:       defaultRoot,
		ListenAddr: defaultListenAddr,
	}

	cmd := &cobra.Command{
		Use:   "cams-hls",
		Short: "Serve the hub's HLS fragments",
		Long: "Serves the playlists, initialisation segments and fMP4 fragments\n" +
			"that cams-rtp2hls writes under --root, plus a small player page.\n" +
			"\n" +
			"NO AUTHORIZATION: anyone who can reach --listen can watch every\n" +
			"camera on this hub and enumerate them. Keep it on loopback, or put\n" +
			"something that authenticates in front of it.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)
			defer cancel()

			return runServer(ctx, cfg)
		},
	}

	cmd.Flags().StringVar(&cfg.ListenAddr, "listen", cfg.ListenAddr,
		"where to accept viewers; there is no authorization, so keep it local")
	cmd.Flags().StringVar(&cfg.Root, "root", cfg.Root,
		"the directory cams-rtp2hls writes under")

	if err := cmd.Execute(); err != nil {
		utils.Logger.Fatal().Err(err).Str("action", "aborting").Msg("hls")
	} else {
		utils.Logger.Info().Str("action", "exiting").Msg("hls")
	}
}

func runServer(ctx context.Context, cfg hlsserver.Config) error {
	srv, err := hlsserver.New(cfg)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	// Shut down on the signal rather than at the end of ListenAndServe, so that
	// a viewer mid-fragment is finished rather than cut off. The grace is
	// bounded: whoever cancels also waits, but not for ever.
	done := make(chan struct{})
	go func() {
		defer close(done)
		<-ctx.Done()

		stopCtx, stop := context.WithTimeout(context.Background(), shutdownGrace)
		defer stop()
		if serr := server.Shutdown(stopCtx); serr != nil {
			utils.Logger.Warn().Err(serr).Msg("hls shutdown")
		}
	}()

	utils.Logger.Info().
		Str("listen", cfg.ListenAddr).
		Str("root", cfg.Root).
		Bool("authorization", false).
		Msg("hls serving")

	err = server.ListenAndServe()
	// A shutdown is how this is meant to end, so it is not a failure.
	if errors.Is(err, http.ErrServerClosed) {
		err = nil
	}
	<-done
	return err
}
