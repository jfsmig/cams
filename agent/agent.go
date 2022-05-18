// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"github.com/aler9/gortsplib"
	"github.com/juju/errors"
	goonvif "github.com/use-go/onvif"
	"log"
	"net"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

type LanAgent struct {
	ScanPeriod  time.Duration
	CheckPeriod time.Duration

	// Last generation number to have been used/
	generation uint32
	// How many generations can be missed before a device is forgotten
	GraceGenerations uint32

	devices    map[string]*OnVifDevice
	interfaces map[string]*LanScanner

	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup
}

func NewLanAgent(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup) *LanAgent {
	return &LanAgent{
		ScanPeriod:  time.Minute,
		CheckPeriod: 30 * time.Second,
		devices:     make(map[string]*OnVifDevice),
		interfaces:  make(map[string]*LanScanner),

		ctx:    ctx,
		cancel: cancel,
		wg:     wg,
	}
}

func (a *LanAgent) Run() {
	defer a.wg.Done()
	defer a.cancel()

	Logger.Info().Str("action", "run").Msg("agent")

	// Discover the local NICs
	itfs, err := net.Interfaces()
	if err != nil {
		log.Panicln(err)
	}
	for _, itf := range itfs {
		if itf.Name != "lo" {
			a.RegisterInterface(itf.Name)
		}
	}

	// Spawn one goroutine per registered interface, for concurrent discoveries
	fn := func(ctx0 context.Context, gen uint32, devs []goonvif.Device) {
		a.LearnSync(ctx0, gen, devs)
	}
	for _, itf := range a.interfaces {
		a.wg.Add(1)
		go itf.RunLoop(a.ctx, a.wg, fn)
	}

	// Run the main loop of the agent that interleaves periodical actions
	// and an eventual clean exit of all the goroutines.
	nextScan := time.After(0)
	nextCheck := time.After(0)
	for {
		select {
		case <-a.ctx.Done():
			Logger.Info().Str("action", "stop").Msg("agent")
			return
		case <-nextScan:
			a.RescanAsync()
			nextScan = time.After(a.ScanPeriod)
		case <-nextCheck:
			a.CheckSync()
			nextCheck = time.After(a.CheckPeriod)
		}
	}
}

func (a *LanAgent) RegisterInterface(itf string) {
	a.interfaces[itf] = NewLanScanner(itf)
	Logger.Info().Str("name", itf).Str("action", "add").Msg("interface")
}

func (a *LanAgent) LearnSingleDeviceSync(ctx context.Context, generation uint32, discovered goonvif.Device) error {
	k := discovered.GetDeviceInfo().SerialNumber

	u := discovered.GetEndpoint("device")
	Logger.Debug().Str("url", u).Uint32("gen", generation).Str("action", "adding").Msg("device")

	parsedUrl, err := url.Parse(u)
	if err != nil {
		return errors.Trace(err)
	}

	if devInPlace, ok := a.devices[k]; ok {
		if generation > devInPlace.generation {
			devInPlace.generation = generation
		}
		return nil
	} else {
		authenticatedDevice, err := goonvif.NewDevice(goonvif.DeviceParams{
			Xaddr:    parsedUrl.Host,
			Username: user,
			Password: password,
		})
		if err != nil {
			return errors.Trace(err)
		}
		transport := gortsplib.TransportUDP
		dev := &OnVifDevice{
			endpoint:    parsedUrl.Host,
			user:        user,
			password:    password,
			generation:  generation,
			onvifClient: authenticatedDevice,
			rtspClient: gortsplib.Client{
				ReadTimeout:           5 * time.Second,
				WriteTimeout:          5 * time.Second,
				RedirectDisable:       true,
				AnyPortEnable:         true,
				Transport:             &transport,
				InitialUDPReadTimeout: 3 * time.Second,
			},
		}
		a.devices[k] = dev
		Logger.Info().
			Str("key", k).
			Str("endpoint", u).
			Str("action", "add").
			Str("user", dev.user).
			Str("password", dev.password).
			Msg("device")
		go dev.RunLoop(ctx, a)
		return nil
	}
}

func (a *LanAgent) LearnSync(ctx context.Context, gen uint32, discovered []goonvif.Device) {
	// Update the devices that match the
	for _, dev := range discovered {
		if err := a.LearnSingleDeviceSync(ctx, gen, dev); err != nil {
			Logger.Warn().Str("url", dev.GetEndpoint("device")).Err(err).Msg("invalid device discovered")
		}
	}
	// Unregister and shut the devices from older generations
	for k, dev := range a.devices {
		if dev.generation < (gen - a.GraceGenerations) {
			delete(a.devices, k)
			dev.Shut()
		}
	}
}

func (a *LanAgent) RescanAsync() {
	gen := atomic.AddUint32(&a.generation, 1)
	Logger.Info().Str("action", "rescan").Uint32("gen", gen).Msg("agent")
	for _, itf := range a.interfaces {
		itf.RescanAsync(a.ctx, gen)
	}
}

func (a *LanAgent) CheckSync() {
	Logger.Info().Str("action", "check").Msg("agent")
}
