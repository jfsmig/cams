// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	utils2 "github.com/jfsmig/cams/go/utils"
	"github.com/jfsmig/go-bags"
	"github.com/jfsmig/onvif/networking"
	"github.com/jfsmig/onvif/sdk"
	"github.com/juju/errors"
	"net/http"
	"regexp"
	"sync"
	"sync/atomic"
	"time"
)

type Agent struct {
	Config AgentConfig

	httpClient http.Client

	// Last generation number to have been used/
	generation uint32

	// How many generations can be missed before a device is forgotten
	GraceGenerations uint32

	singletonLock sync.Mutex
	dataLock      sync.Mutex

	devices    bags.SortedObj[string, *Camera]
	interfaces bags.SortedObj[string, *Nic]

	// Fields extracted from the configuration
	devicesStatic              []CameraConfig
	interfacesStatic           []string
	interfacesDiscoverPatterns []string

	nicsGroup utils2.Swarm
	camsSwarm utils2.Swarm
}

func NewLanAgent(cfg AgentConfig) *Agent {
	lan := &Agent{
		Config:     cfg,
		httpClient: http.Client{},

		devices:    make([]*Camera, 0),
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

	return lan
}

// UpdateStreamExpectation implements a
func (lan *Agent) UpdateStreamExpectation(camId string, cmd CamCommand) error {
	// Locate the camera
	cam := func(camId string) *Camera {
		lan.dataLock.Lock()
		defer lan.dataLock.Unlock()
		cam, _ := lan.devices.Get(camId)
		return cam
	}(camId)

	if cam == nil {
		return errors.New("no such camera")
	}

	switch cmd {
	case CamCommandPlay:
		if cam.State == CamAgentOff {
			lan.camsSwarm.Run(runCam(cam))
		}
		cam.PlayStream()
		return nil
	case CamCommandPause:
		cam.StopStream()
		return nil
	default:
		return errors.New("BUG: unexpected command")
	}
}

func (lan *Agent) Run(ctx context.Context) {
	if !lan.singletonLock.TryLock() {
		panic("BUG: the LAN agent coroutine is a singleton")
	}
	defer lan.singletonLock.Unlock()

	utils2.Logger.Info().Str("action", "start").Msg("lan")

	// Cameras may come ang go, so a simple goroutine swarm if enough.
	lan.camsSwarm = utils2.NewSwarm(ctx)
	defer lan.camsSwarm.Cancel()

	// ... This is not the case for network interfaces that are rather stable.
	lan.nicsGroup = utils2.NewGroup(ctx)
	defer lan.nicsGroup.Cancel()

	// Perform a first discovery of the local interfaces.
	// No need to do it periodically, interfaces are unlikely plug & play
	if err := lan.discoverNics(); err != nil {
		utils2.Logger.Error().Err(err).Msg("disc")
		return
	}

	// Spawn one goroutine per registered interface, for concurrent discoveries
	fn := func(ctx0 context.Context, gen uint32, devs []networking.ClientInfo) {
		lan.learnAllCamerasSync(ctx0, gen, devs)
	}
	for _, itf := range lan.interfaces {
		func(itf *Nic) {
			lan.nicsGroup.Run(func(c context.Context) { itf.RunRescanLoop(c, fn) })
		}(itf)
	}

	lan.nicsGroup.Run(func(c context.Context) { lan.runTimers(c) })

	utils2.Logger.Info().Str("action", "wait nics").Msg("lan")

	// Wait for the discovery goroutines to stop, this will happen until a strong
	// error condition occurs.
	lan.nicsGroup.Wait()

	utils2.Logger.Info().Str("action", "wait cams").Msg("lan")

	// Then ensure no camera goroutine is leaked running in the background
	lan.camsSwarm.Cancel()
	lan.camsSwarm.Wait()
}

func (lan *Agent) Cameras() []*Camera {
	lan.dataLock.Lock()
	defer lan.dataLock.Unlock()
	out := make([]*Camera, 0, len(lan.devices))
	for _, cam := range lan.devices {
		out = append(out, cam)
	}
	return out
}

// runTimers runs the main loop of the agent to trigger periodical actions
func (lan *Agent) runTimers(ctx context.Context) {
	nextScan := time.After(0)
	nextCheck := time.After(0)
	for {
		select {
		case <-ctx.Done():
			utils2.Logger.Info().Str("action", "stop").Msg("lan")
			return
		case <-nextScan:
			lan.triggerRescanAsync(ctx)
			nextScan = time.After(lan.Config.GetScanPeriod())
		case <-nextCheck:
			HttpClient.CloseIdleConnections()
			utils2.Logger.Info().
				Str("action", "check").
				Int("devices", len(lan.devices)).
				Int("interfaces", len(lan.interfaces)).
				Msg("lan")
			nextCheck = time.After(lan.Config.GetCheckPeriod())
		}
	}
}

// discoverNics does the discovery from the output of a given function.
// It helps to test the logic.
func (lan *Agent) discoverNics() error {
	itfs, err := utils2.DiscoverSystemNics()
	if err != nil {
		return errors.Trace(err)
	}

	utils2.Logger.Trace().Strs("interfaces", itfs).Msg("disc")

	for _, itf := range itfs {
		lan.maybeRegisterInterface(itf)
	}

	for _, itf := range lan.interfacesStatic {
		utils2.Logger.Info().Str("itf", itf).Str("action", "force").Msg("disc")
		lan.registerInterface(itf)
	}
	return nil
}

func (lan *Agent) maybeRegisterInterface(itf string) {
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
			utils2.Logger.Warn().Str("pattern", pattern0).Str("itf", itf).Err(err).Msg("disc")
		} else if !match {
			continue
		} else if !not {
			utils2.Logger.Info().Str("pattern", pattern0).Str("itf", itf).Str("action", "add").Msg("disc")
			lan.registerInterface(itf)
		} else {
			utils2.Logger.Debug().Str("pattern", pattern0).Str("itf", itf).Str("action", "skip").Msg("disc")
		}
		return
	}
}

func (lan *Agent) registerInterface(itf string) {
	lan.interfaces.Add(NewNIC(itf))
}

func (lan *Agent) learnSingleCameraSync(ctx context.Context, generation uint32, discovered networking.ClientInfo) error {
	// Preliminary check of the existence of the camera, before starting expensive queries
	lan.dataLock.Lock()
	devInPlace, already := lan.devices.Get(discovered.Uuid)
	if already && generation > devInPlace.generation {
		devInPlace.generation = generation
	}
	lan.dataLock.Unlock()
	if already {
		return nil
	}

	appliance, err := sdk.NewDevice(ctx, discovered, networking.ClientAuth{
		Username: User,
		Password: Password,
	}, &HttpClient)
	if err != nil {
		return errors.Trace(err)
	}

	// Here come the http requests
	dev := NewCamera(lan, appliance)

	lan.dataLock.Lock()
	defer lan.dataLock.Unlock()

	// If the camera is already know, it's very unlikely, but it may happen in case of a stale discovery,
	// let's just update its generation counter
	devInPlace, already = lan.devices.Get(dev.PK())
	if already {
		if generation > devInPlace.generation {
			devInPlace.generation = generation
		} // else ... stale discovery (why not?)
	} else {
		dev.generation = generation
		lan.devices.Add(dev)
		utils2.Logger.Info().
			Str("key", dev.PK()).
			Str("endpoint", discovered.Xaddr).
			Uint32("gen", generation).
			Str("action", "add").
			Msg("device")

		lan.camsSwarm.Run(runCam(dev))
	}
	return nil
}

func (lan *Agent) learnAllCamerasSync(ctx context.Context, gen uint32, discovered []networking.ClientInfo) {
	// Update the devices that match the
	for _, dev := range discovered {
		if err := lan.learnSingleCameraSync(ctx, gen, dev); err != nil {
			utils2.Logger.Warn().Str("url", dev.Xaddr).Err(err).Msg("invalid device discovered")
		}
	}

	toBePurged := lan.camsToBePurged(gen)
	if len(toBePurged) > 0 {
		utils2.Logger.Info().Str("action", "purge").Interface("count", len(toBePurged)).Msg("lan")
	}

	for _, dev := range toBePurged {
		lan.dataLock.Lock()
		lan.devices.Remove(dev.PK())
		dev.StopStream()
		lan.dataLock.Unlock()
	}
}

func (lan *Agent) camsToBePurged(gen uint32) []*Camera {
	out := make([]*Camera, 0)

	lan.dataLock.Lock()
	defer lan.dataLock.Unlock()

	// Unregister and shut the devices from older generations
	for i := len(lan.devices); i > 0; i-- {
		dev := lan.devices[i-1]
		if delta(gen, dev.generation) < lan.GraceGenerations {
			out = append(out, dev)
		}
	}

	return out
}

type Unsigned interface {
	uint | uint32 | uint16 | uint8
}

func maxValue[T Unsigned]() T {
	var zero T
	return zero - 1
}

func delta[T Unsigned](hi, lo T) T {
	if hi >= lo {
		return hi - lo
	} else {
		return hi - lo + maxValue[T]()
	}
}

func (lan *Agent) triggerRescanAsync(ctx context.Context) {
	gen := atomic.AddUint32(&lan.generation, 1)
	for _, itf := range lan.interfaces {
		itf.TriggerRescanAsync(ctx, gen)
	}
}

func (lan *Agent) PK() string { return "lan" }
