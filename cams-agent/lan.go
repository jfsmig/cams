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
	swarm utils.Swarm

	ScanPeriod  time.Duration
	CheckPeriod time.Duration

	// Last generation number to have been used/
	generation uint32

	// How many generations can be missed before a device is forgotten
	GraceGenerations uint32

	lock sync.Mutex

	devices    bags.SortedObj[string, *LanCamera]
	interfaces bags.SortedObj[string, *Nic]
	observers  bags.SortedObj[string, CameraObserver]

	// Fields extracted from the configuration
	devicesStatic              []CameraConfig
	interfacesStatic           []string
	interfacesDiscoverPatterns []string
}

type CameraState uint32

const (
	CameraOnline CameraState = iota
	CameraOffline
)

type CameraObserver interface {
	PK() string
	UpdateCamera(camId string, state CameraState)
}

func NewLanAgent(cfg AgentConfig) *lanAgent {
	lan := &lanAgent{
		ScanPeriod:  5 * time.Second,
		CheckPeriod: 30 * time.Second,

		devices:    make([]*LanCamera, 0),
		interfaces: make([]*Nic, 0),

		interfacesDiscoverPatterns: []string{},
		interfacesStatic:           []string{},
		devicesStatic:              []CameraConfig{},
	}

	for _, itf := range cfg.Interfaces {
		lan.registerInterface(itf)
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

	return lan
}

func (lan *lanAgent) AttachObserver(observer CameraObserver) {
	lan.observers.Add(observer)
}

func (lan *lanAgent) DetachObserver(observer CameraObserver) {
	lan.observers.Remove(observer.PK())
}

func (lan *lanAgent) Notify(camId string, state CameraState) {
	for _, observer := range lan.observers {
		observer.UpdateCamera(camId, state)
	}
}

func (lan *lanAgent) Run(ctx context.Context) {
	s := utils.NewSwarm(ctx)
	defer s.Wait()
	defer s.Cancel()

	utils.Logger.Info().Str("action", "start").Msg("lan")

	// Perform a first discovery of the local interfaces.
	// No need to do it periodically, interfaces are unlikely plug & play
	if err := lan.discoverNics(); err != nil {
		utils.Logger.Error().Err(err).Msg("discovery")
		return
	}

	// Spawn one goroutine per registered interface, for concurrent discoveries
	fn := func(ctx0 context.Context, gen uint32, devs []goonvif.Device) {
		lan.learnSync(ctx0, gen, devs)
	}
	for _, itf := range lan.interfaces {
		s.Run(func(c context.Context) { itf.RunRescanLoop(c, fn) })
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
			lan.triggerRescanAsync(ctx)
			nextScan = time.After(lan.ScanPeriod)
		}
	}
}

func (lan *lanAgent) GetCamera(camId string) *LanCamera {
	lan.lock.Lock()
	defer lan.lock.Lock()
	cam, _ := lan.devices.Get(camId)
	return cam
}

// discoverNics does the discovery from the output of a given function.
// It helps to test the logic.
func (lan *lanAgent) discoverNics() error {
	itfs, err := discoverSystemNics()
	if err != nil {
		return errors.Trace(err)
	}

	utils.Logger.Trace().Strs("interfaces", itfs).Msg("discovery")

	for _, itf := range itfs {
		lan.maybeRegisterInterface(itf)
	}

	for _, itf := range lan.interfacesStatic {
		utils.Logger.Info().Str("itf", itf).Str("action", "force").Msg("discovery")
		lan.registerInterface(itf)
	}
	return nil
}

func (lan *lanAgent) maybeRegisterInterface(itf string) {
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
			lan.registerInterface(itf)
		} else {
			utils.Logger.Debug().Str("pattern", pattern0).Str("itf", itf).Str("action", "skip").Msg("discovery")
		}
		return
	}
}

func (lan *lanAgent) registerInterface(itf string) {
	lan.interfaces.Add(NewNIC(itf))
}

func (lan *lanAgent) learnSingleDeviceSync(ctx context.Context, generation uint32, discovered goonvif.Device) error {
	k := discovered.GetDeviceInfo().SerialNumber

	u := discovered.GetEndpoint("device")
	utils.Logger.Debug().Str("url", u).Uint32("gen", generation).Str("action", "adding").Msg("device")

	parsedUrl, err := url.Parse(u)
	if err != nil {
		return errors.Trace(err)
	}

	lan.lock.Lock()
	defer lan.lock.Lock()

	// If the camera is already know, let's just update its generation counter
	devInPlace, ok := lan.devices.Get(k)
	if ok {
		if generation > devInPlace.generation {
			devInPlace.generation = generation
		}
		return nil
	}

	// If not, the camera is new on the LAN
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

	lan.Notify(dev.PK(), CameraOnline)
	return nil
}

func (lan *lanAgent) learnSync(ctx context.Context, gen uint32, discovered []goonvif.Device) {
	// UpdateCamera the devices that match the
	for _, dev := range discovered {
		if err := lan.learnSingleDeviceSync(ctx, gen, dev); err != nil {
			utils.Logger.Warn().Str("url", dev.GetEndpoint("device")).Err(err).Msg("invalid device discovered")
		}
	}

	lan.lock.Lock()
	defer lan.lock.Unlock()

	// Unregister and shut the devices from older generations
	for i := len(lan.devices); i > 0; i-- {
		dev := lan.devices[i-1]
		if dev.generation < (gen - lan.GraceGenerations) {
			lan.devices.Remove(dev.PK())
			lan.Notify(dev.PK(), CameraOffline)
			dev.StopStream()
		}
	}
}

func (lan *lanAgent) triggerRescanAsync(ctx context.Context) {
	gen := atomic.AddUint32(&lan.generation, 1)
	utils.Logger.Info().Str("action", "rescan").Uint32("gen", gen).Msg("lan")
	for _, itf := range lan.interfaces {
		itf.TriggerRescanAsync(ctx, gen)
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
