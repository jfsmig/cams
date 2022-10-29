// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"github.com/jfsmig/cams/utils"
	"github.com/jfsmig/go-bags"
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

type lanAgent struct {
	ScanPeriod  time.Duration
	CheckPeriod time.Duration

	// Last generation number to have been used/
	generation uint32

	// How many generations can be missed before a device is forgotten
	GraceGenerations uint32

	devices    bags.SortedObj[string, *LanCamera]
	interfaces bags.SortedObj[string, *Nic]

	// Fields extracted from the configuration
	devicesStatic              []CameraConfig
	interfacesStatic           []string
	interfacesDiscoverPatterns []string
}

func NewLanAgent() *lanAgent {
	return &lanAgent{
		ScanPeriod:  5 * time.Second,
		CheckPeriod: 30 * time.Second,

		devices:    make([]*LanCamera, 0),
		interfaces: make([]*Nic, 0),

		interfacesDiscoverPatterns: []string{},
		interfacesStatic:           []string{},
		devicesStatic:              []CameraConfig{},
	}
}

func (lan *lanAgent) Configure(cfg AgentConfig) {
	for _, itf := range cfg.Interfaces {
		lan.RegisterInterface(itf)
	}

	lan.interfacesDiscoverPatterns = cfg.DiscoverPatterns
	lan.interfacesStatic = cfg.Interfaces
	lan.devicesStatic = cfg.Cameras

	if cfg.CheckPeriod > 0 {
		lan.CheckPeriod = time.Duration(cfg.CheckPeriod) * time.Second
	}
	if cfg.ScanPeriod > 0 {
		lan.ScanPeriod = time.Duration(cfg.ScanPeriod) * time.Second
	}
}

func discoverSystemNics() ([]string, error) {
	var out []string
	itfs, err := net.Interfaces()
	if err != nil {
		return nil, errors.Trace(err)
	}
	for _, itf := range itfs {
		out = append(out, itf.Name)
	}
	return out, nil
}

// DiscoverNics does the discovery from the output of a given function.
// It helps to test the logic.
func (lan *lanAgent) DiscoverNics() error {
	itfs, err := discoverSystemNics()
	if err != nil {
		return errors.Trace(err)
	}

	utils.Logger.Trace().Strs("interfaces", itfs).Msg("discovery")

	for _, itf := range itfs {
		lan.MaybeRegisterInterface(itf)
	}

	for _, itf := range lan.interfacesStatic {
		utils.Logger.Info().Str("itf", itf).Str("action", "force").Msg("discovery")
		lan.RegisterInterface(itf)
	}
	return nil
}

func (lan *lanAgent) MaybeRegisterInterface(itf string) {
	for _, pattern0 := range lan.interfacesDiscoverPatterns {
		if len(pattern0) < 2 {
			continue
		}
		pattern := pattern0
		not := pattern[0] == '!'
		if not {
			pattern = pattern[1:]
		}
		if match, err := regexp.MatchString(pattern, itf); err != nil {
			utils.Logger.Warn().Str("pattern", pattern0).Str("itf", itf).Err(err).Msg("discovery")
		} else if !match {
			continue
		} else if !not {
			utils.Logger.Info().Str("pattern", pattern0).Str("itf", itf).Str("action", "add").Msg("discovery")
			lan.RegisterInterface(itf)
		} else {
			utils.Logger.Debug().Str("pattern", pattern0).Str("itf", itf).Str("action", "skip").Msg("discovery")
		}
		return
	}
}

func (lan *lanAgent) RegisterInterface(itf string) {
	lan.interfaces.Add(NewNIC(itf))
}

func (lan *lanAgent) Run(ctx0 context.Context) {
	wg := sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(ctx0)
	defer cancel()

	utils.Logger.Info().Str("action", "run").Msg("lan")

	// Perform a first discovery of the local interfaces.
	// No need to do it periodically, interfaces are unlikely plug & play
	if err := lan.DiscoverNics(); err != nil {
		utils.Logger.Error().Err(err).Msg("discovery")
		return
	}

	// Spawn one goroutine per registered interface, for concurrent discoveries
	fn := func(ctx0 context.Context, gen uint32, devs []goonvif.Device) {
		lan.LearnSync(ctx0, gen, devs)
	}
	for _, itf := range lan.interfaces {
		wg.Add(1)
		go swarmRun(ctx, cancel, &wg, func(c context.Context) { itf.RunRescanLoop(c, fn) })
	}

	// Run the main loop of the agent that interleaves periodical actions
	// and an eventual clean exit of all the goroutines.
	nextScan := time.After(0)
	for {
		select {
		case <-ctx.Done():
			utils.Logger.Info().Str("action", "stop").Msg("lan")
			return
		case <-nextScan:
			lan.TriggerRescanAsync(ctx)
			nextScan = time.After(lan.ScanPeriod)
		}
	}
}

func (lan *lanAgent) LearnSingleDeviceSync(ctx context.Context, generation uint32, discovered goonvif.Device) error {
	k := discovered.GetDeviceInfo().SerialNumber

	u := discovered.GetEndpoint("device")
	utils.Logger.Debug().Str("url", u).Uint32("gen", generation).Str("action", "adding").Msg("device")

	parsedUrl, err := url.Parse(u)
	if err != nil {
		return errors.Trace(err)
	}

	if devInPlace, ok := lan.devices.Get(k); ok {
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
		dev := &LanCamera{
			ID:          k,
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
		lan.devices.Add(dev)
		utils.Logger.Info().
			Str("key", dev.PK()).
			Str("endpoint", u).
			Str("action", "add").
			Str("user", dev.user).
			Str("password", dev.password).
			Msg("device")
		go dev.RunLoop(ctx, lan)
		return nil
	}
}

func (lan *lanAgent) LearnSync(ctx context.Context, gen uint32, discovered []goonvif.Device) {
	// Update the devices that match the
	for _, dev := range discovered {
		if err := lan.LearnSingleDeviceSync(ctx, gen, dev); err != nil {
			utils.Logger.Warn().Str("url", dev.GetEndpoint("device")).Err(err).Msg("invalid device discovered")
		}
	}
	// Unregister and shut the devices from older generations
	for i := len(lan.devices); i > 0; i-- {
		dev := lan.devices[i-1]
		if dev.generation < (gen - lan.GraceGenerations) {
			lan.devices.Remove(dev.PK())
			dev.StopStream()
		}
	}
}

func (lan *lanAgent) TriggerRescanAsync(ctx context.Context) {
	gen := atomic.AddUint32(&lan.generation, 1)
	utils.Logger.Info().Str("action", "rescan").Uint32("gen", gen).Msg("lan")
	for _, itf := range lan.interfaces {
		itf.TriggerRescanAsync(ctx, gen)
	}
}
