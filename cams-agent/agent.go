// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"github.com/jfsmig/cams/utils"
	"net"
	"net/url"
	"regexp"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aler9/gortsplib"
	"github.com/juju/errors"
	goonvif "github.com/use-go/onvif"
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

type DiscoveryFunc func() ([]string, error)

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

func (lan *LanAgent) Configure(cfg AgentConfig) {
	for _, itf := range cfg.Interfaces {
		lan.RegisterInterface(itf)
	}
	if len(cfg.DiscoverPatterns) <= 0 {
		lan.Discover(cfg.DiscoverPatterns...)
	}
	if cfg.CheckPeriod > 0 {
		lan.CheckPeriod = time.Duration(cfg.CheckPeriod) * time.Second
	}
	if cfg.ScanPeriod > 0 {
		lan.ScanPeriod = time.Duration(cfg.ScanPeriod) * time.Second
	}
}

func realDiscovery() ([]string, error) {
	var out []string
	itfs, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, itf := range itfs {
		out = append(out, itf.Name)
	}
	return out, nil
}

// Discover performs a discovery of the local NICs
func (lan *LanAgent) Discover(patterns ...string) error {
	return lan.DiscoverFrom(realDiscovery, patterns...)
}

// DiscoverFrom does the discovery from the output of a given function.
// It helps to test the logic.
func (lan *LanAgent) DiscoverFrom(source DiscoveryFunc, patterns ...string) error {
	itfs, err := source()
	if err != nil {
		return err
	}
	for _, itf := range itfs {
		for _, pattern := range patterns {
			if len(pattern) < 2 {
				continue
			}
			not := pattern[0] == '!'
			if not {
				pattern = pattern[1:]
			}
			if match, err := regexp.MatchString(pattern, itf); err != nil {
				utils.Logger.Warn().Err(err).Str("itf", itf).Msg("interface matching")
			} else if match && !not {
				lan.RegisterInterface(itf)
			} else {
				utils.Logger.Info().Str("itf", itf).Msg("interface skipped")
			}
		}
	}
	return nil
}

func (lan *LanAgent) RegisterInterface(itf string) {
	lan.interfaces[itf] = NewLanScanner(itf)
	utils.Logger.Info().Str("name", itf).Str("action", "add").Msg("interface")
}

func (lan *LanAgent) Run() {
	defer lan.wg.Done()
	defer lan.cancel()

	utils.Logger.Info().Str("action", "run").Msg("agent")

	// Spawn one goroutine per registered interface, for concurrent discoveries
	fn := func(ctx0 context.Context, gen uint32, devs []goonvif.Device) {
		lan.LearnSync(ctx0, gen, devs)
	}
	for _, itf := range lan.interfaces {
		lan.wg.Add(1)
		go itf.RunLoop(lan.ctx, lan.wg, fn)
	}

	// Run the main loop of the agent that interleaves periodical actions
	// and an eventual clean exit of all the goroutines.
	nextScan := time.After(0)
	nextCheck := time.After(0)
	for {
		select {
		case <-lan.ctx.Done():
			utils.Logger.Info().Str("action", "stop").Msg("agent")
			return
		case <-nextScan:
			lan.RescanAsync()
			nextScan = time.After(lan.ScanPeriod)
		case <-nextCheck:
			lan.CheckSync()
			nextCheck = time.After(lan.CheckPeriod)
		}
	}
}

func (lan *LanAgent) LearnSingleDeviceSync(ctx context.Context, generation uint32, discovered goonvif.Device) error {
	k := discovered.GetDeviceInfo().SerialNumber

	u := discovered.GetEndpoint("device")
	utils.Logger.Debug().Str("url", u).Uint32("gen", generation).Str("action", "adding").Msg("device")

	parsedUrl, err := url.Parse(u)
	if err != nil {
		return errors.Trace(err)
	}

	if devInPlace, ok := lan.devices[k]; ok {
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
		lan.devices[k] = dev
		utils.Logger.Info().
			Str("key", k).
			Str("endpoint", u).
			Str("action", "add").
			Str("user", dev.user).
			Str("password", dev.password).
			Msg("device")
		go dev.RunLoop(ctx, lan)
		return nil
	}
}

func (lan *LanAgent) LearnSync(ctx context.Context, gen uint32, discovered []goonvif.Device) {
	// Update the devices that match the
	for _, dev := range discovered {
		if err := lan.LearnSingleDeviceSync(ctx, gen, dev); err != nil {
			utils.Logger.Warn().Str("url", dev.GetEndpoint("device")).Err(err).Msg("invalid device discovered")
		}
	}
	// Unregister and shut the devices from older generations
	for k, dev := range lan.devices {
		if dev.generation < (gen - lan.GraceGenerations) {
			delete(lan.devices, k)
			dev.Shut()
		}
	}
}

func (lan *LanAgent) RescanAsync() {
	gen := atomic.AddUint32(&lan.generation, 1)
	utils.Logger.Info().Str("action", "rescan").Uint32("gen", gen).Msg("agent")
	for _, itf := range lan.interfaces {
		itf.RescanAsync(lan.ctx, gen)
	}
}

func (lan *LanAgent) CheckSync() {
	utils.Logger.Info().Str("action", "check").Msg("agent")
}
