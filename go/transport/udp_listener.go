package transport

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/juju/errors"
	"golang.org/x/sync/errgroup"
)

type UdpListener struct {
	media   net.PacketConn
	control net.PacketConn

	singletonLock sync.Mutex
}

func (ul *UdpListener) GetPortMedia() int {
	return getPort(ul.media)
}

func (ul *UdpListener) GetPortControl() int {
	return getPort(ul.control)
}

func (ul *UdpListener) OpenRandom(addr string) error {
	if ul == nil || ul.media != nil || ul.control != nil {
		return errors.New("opening an opened listener")
	}
	if !ul.singletonLock.TryLock() {
		return errors.New("opening a running listener")
	}
	defer ul.release()

	var err error
	ul.media, err = openPort(addr, 0)
	if err != nil {
		return errors.Trace(err)
	}

	ul.control, err = openPort(addr, 0)
	if err != nil {
		ul.media.Close()
		return errors.Trace(err)
	}

	return nil
}

func (ul *UdpListener) OpenPair(addr string) error {
	if ul == nil || ul.media != nil || ul.control != nil {
		return errors.New("opening an opened listener")
	}
	if !ul.singletonLock.TryLock() {
		return errors.New("opening a running listener")
	}
	defer ul.release()

	var err error
	for port := 0; ; port++ {
		ul.media, err = openPort(addr, port)
		if err != nil {
			return errors.Trace(err)
		}
		if port == 0 {
			port = getPort(ul.media)
		}

		ul.control, err = openPort(addr, port+1)
		if err == nil {
			return nil
		}

		ul.media.Close()
	}
}

func (ul *UdpListener) Close() {
	if ul == nil || ul.singletonLock.TryLock() {
		return
	}
	defer ul.release()

	if ul.media != nil {
		ul.media.Close()
		ul.media = nil
	}
	if ul.control != nil {
		ul.control.Close()
		ul.control = nil
	}
}

func (ul *UdpListener) Run(ctx context.Context, outMedia, outControl chan []byte) error {
	if err := ul.acquire(); err != nil {
		return errors.Trace(err)
	}
	defer ul.release()

	g, ctx := errgroup.WithContext(ctx)
	getBuffer := func() []byte { return make([]byte, 8192) }

	runCnx := func(cnx net.PacketConn, out chan []byte) error {
		for ctx.Err() == nil {
			buf := getBuffer()
			_ = cnx.SetDeadline(time.Now().Add(time.Second))
			if n, _, err := cnx.ReadFrom(buf); err != nil {
				if err, ok := err.(net.Error); ok && err.Timeout() {
					continue
				}
				return errors.Trace(err)
			} else {
				out <- buf[:n]
			}
		}
		return nil
	}
	g.Go(func() error { return runCnx(ul.media, outMedia) })
	g.Go(func() error { return runCnx(ul.control, outControl) })
	return g.Wait()
}

func (ul *UdpListener) acquire() error {
	if ul == nil || ul.media == nil || ul.control == nil {
		return errors.New("udp listener invalid state")
	}
	if !ul.singletonLock.TryLock() {
		log.Panicln("here", ul, ul.media, ul.control)
		return errors.New("udp listener running")
	}
	return nil
}

func (ul *UdpListener) release() {
	if ul != nil {
		ul.singletonLock.Unlock()
	}
}

func openPort(addr string, port int) (net.PacketConn, error) {
	cnx, err := net.ListenPacket("udp", fmt.Sprintf("%s:%d", addr, port))
	return cnx, errors.Trace(err)
}

func getPort(cnx net.PacketConn) int {
	if cnx == nil {
		return -1
	}
	return cnx.LocalAddr().(*net.UDPAddr).Port
}
