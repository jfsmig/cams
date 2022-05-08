package main

import (
	"context"
	"github.com/aler9/gortsplib"
	goonvif "github.com/use-go/onvif"
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
}

func NewLanAgent() *LanAgent {
	return &LanAgent{
		ScanPeriod:  time.Minute,
		CheckPeriod: 30 * time.Second,
		devices:     make(map[string]*OnVifDevice),
		interfaces:  make(map[string]*LanScanner),
	}
}

func (a *LanAgent) RunLoop(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	Logger.Info().Str("action", "run").Msg("agent")

	// Spawn one goroutine per registered interface, for concurrent discoveries
	fn := func(ctx0 context.Context, gen uint32, devs []goonvif.Device) {
		a.LearnSync(ctx0, gen, devs)
	}
	for _, itf := range a.interfaces {
		wg.Add(1)
		go itf.RunLoop(ctx, wg, fn)
	}

	// Run the main loop of the agent that interleaves periodical actions
	// and an eventual clean exit of all the goroutines.
	nextScan := time.After(0)
	nextCheck := time.After(0)
	for {
		select {
		case <-ctx.Done():
			Logger.Info().Str("action", "stop").Msg("agent")
			return
		case <-nextScan:
			a.RescanAsync(ctx)
			nextScan = time.After(a.ScanPeriod)
		case <-nextCheck:
			a.CheckSync(ctx)
			nextCheck = time.After(a.CheckPeriod)
		}
	}
}

func (a *LanAgent) RegisterInterface(itf string) {
	a.interfaces[itf] = NewLanScanner(itf)
	Logger.Info().Str("name", itf).Str("action", "add").Msg("interface")
}

func (a *LanAgent) LearnSingleDeviceSync(ctx context.Context, generation uint32, discovered goonvif.Device) {
	u := discovered.GetEndpoint("device")
	if devInPlace, ok := a.devices[u]; ok {
		if generation > devInPlace.generation {
			devInPlace.generation = generation
		}
	} else {
		transport := gortsplib.TransportUDP
		dev := &OnVifDevice{
			generation: generation,
			onvifURL:   u,
			dev:        discovered,
			client: gortsplib.Client{
				ReadTimeout:           5 * time.Second,
				WriteTimeout:          5 * time.Second,
				RedirectDisable:       true,
				AnyPortEnable:         true,
				Transport:             &transport,
				InitialUDPReadTimeout: 3 * time.Second,
			},
		}
		a.devices[u] = dev
		Logger.Info().Str("url", u).Str("action", "add").Msg("device")
		go dev.RunLoop(ctx, a)
	}
}

func (a *LanAgent) LearnSync(ctx context.Context, gen uint32, discovered []goonvif.Device) {
	// Update the devices that match the
	for _, dev := range discovered {
		a.LearnSingleDeviceSync(ctx, gen, dev)
	}
	// Unregister and shut the devices from older generations
	for k, dev := range a.devices {
		if dev.generation < (gen - a.GraceGenerations) {
			delete(a.devices, k)
			dev.Shut()
		}
	}
}

func (a *LanAgent) RescanAsync(ctx context.Context) {
	gen := atomic.AddUint32(&a.generation, 1)
	Logger.Info().Str("action", "rescan").Uint32("gen", gen).Msg("agent")
	for _, itf := range a.interfaces {
		itf.RescanAsync(ctx, gen)
	}
}

func (a *LanAgent) CheckSync(ctx context.Context) {
	Logger.Info().Str("action", "check").Msg("agent")
}
