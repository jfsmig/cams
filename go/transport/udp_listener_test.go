package transport

import (
	"context"
	"golang.org/x/sync/errgroup"
	"testing"
)

func TestUdpListener_doubleOps(t *testing.T) {
	ul := UdpListener{}
	if err := ul.OpenRandom("0.0.0.0"); err != nil {
		t.Fatal(err)
	}

	// double open must fail
	if err := ul.OpenRandom("0.0.0.0"); err == nil {
		t.Fatal("unexpected success")
	}

	ul.Close()
	ul.Close()

	// open must succeed after a close
	if err := ul.OpenRandom("0.0.0.0"); err == nil {
		t.Fatal("unexpected success")
	}
	defer ul.Close()
	// double open must still fail
	if err := ul.OpenRandom("0.0.0.0"); err == nil {
		t.Fatal("unexpected success")
	}
}

func TestUdpListener_OpenRandom(t *testing.T) {
	ul := UdpListener{}
	if err := ul.OpenRandom("0.0.0.0"); err != nil {
		t.Fatal(err)
	}
	defer ul.Close()

	if ul.GetPortMedia() <= 0 {
		t.Fatal("no media port")
	}
	if ul.GetPortControl() <= 0 {
		t.Fatal("no control port")
	}

	outMedia := make(chan []byte, 1)
	outControl := make(chan []byte, 1)

	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return ul.Run(ctx, outMedia, outControl)
	})
	cancel()

	if err := g.Wait(); err != nil {
		t.Fatal("unexpected error, no callback returned an error")
	}
}
