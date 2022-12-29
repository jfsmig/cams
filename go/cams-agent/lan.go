// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"regexp"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jfsmig/go-bags"
	"github.com/jfsmig/onvif/sdk"
	"github.com/juju/errors"

	"github.com/jfsmig/cams/go/utils"
)

type lanAgent struct {
	ScanPeriod  time.Duration
	CheckPeriod time.Duration

	// Last generation number to have been used/
	generation uint32

	// How many generations can be missed before a device is forgotten
	GraceGenerations uint32

	singletonLock sync.Mutex
	dataLock      sync.Mutex

	devices    bags.SortedObj[string, *LanCamera]
	interfaces bags.SortedObj[string, *Nic]
	observers  bags.SortedObj[string, CameraObserver]

	// Fields extracted from the configuration
	devicesStatic              []CameraConfig
	interfacesStatic           []string
	interfacesDiscoverPatterns []string

	nicsGroup utils.Swarm
	camsSwarm utils.Swarm
}

type CameraState uint32

const (
	CameraState_Online CameraState = iota
	CameraState_Offline
)

type CameraObserver interface {
	PK() string
	UpdateCameraState(camId string, state CameraState)
}

func NewLanAgent(cfg AgentConfig) *lanAgent {
	lan := &lanAgent{
		ScanPeriod:  1 * time.Minute,
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

func (lan *lanAgent) AttachCameraObserver(observer CameraObserver) {
	lan.observers.Add(observer)
}

func (lan *lanAgent) DetachCameraObserver(observer CameraObserver) {
	lan.observers.Remove(observer.PK())
}

func (lan *lanAgent) UpdateStreamExpectation(camId string, cmd StreamExpectation) {
	// Locate the camera
	cam := func(camId string) *LanCamera {
		lan.dataLock.Lock()
		defer lan.dataLock.Unlock()
		cam, _ := lan.devices.Get(camId)
		return cam
	}(camId)

	if cam == nil {
		utils.Logger.Info().Str("cam", camId).Interface("cmd", cmd).Err(errors.New("cam not found")).Msg("lan")
		return
	}

	switch cmd {
	case UpstreamAgent_ExpectPlay:
		if cam.State == CamAgentOff {
			lan.camsSwarm.Run(runCam(cam))
		}
		cam.PlayStream()
	case UpstreamAgent_ExpectPause:
		cam.StopStream()
	}
}

func (lan *lanAgent) Notify(camId string, state CameraState) {
	for _, observer := range lan.observers {
		observer.UpdateCameraState(camId, state)
	}
}

func (lan *lanAgent) Run(ctx context.Context) {
	if !lan.singletonLock.TryLock() {
		panic("BUG: the LAN agent coroutine is a singleton")
	}
	defer lan.singletonLock.Unlock()

	utils.Logger.Info().Str("action", "start").Msg("lan")

	// Cameras may come ang go, so a simple goroutine swarm if enough.
	lan.camsSwarm = utils.NewSwarm(ctx)
	defer lan.camsSwarm.Cancel()

	// ... This is not the case for network interfaces that are rather stable.
	lan.nicsGroup = utils.NewGroup(ctx)
	defer lan.nicsGroup.Cancel()

	// Perform a first discovery of the local interfaces.
	// No need to do it periodically, interfaces are unlikely plug & play
	if err := lan.discoverNics(); err != nil {
		utils.Logger.Error().Err(err).Msg("disc")
		return
	}

	// Spawn one goroutine per registered interface, for concurrent discoveries
	fn := func(ctx0 context.Context, gen uint32, devs []sdk.Appliance) {
		lan.learnAllCamerasSync(ctx0, gen, devs)
	}
	for _, itf := range lan.interfaces {
		func(itf *Nic) {
			lan.nicsGroup.Run(func(c context.Context) { itf.RunRescanLoop(c, fn) })
		}(itf)
	}

	lan.nicsGroup.Run(func(c context.Context) { lan.RunTimers(c) })

	utils.Logger.Info().Str("action", "wait nics").Msg("lan")

	// Wait for the discovery goroutines to stop, this will happen until a strong
	// error condition occurs.
	lan.nicsGroup.Wait()

	utils.Logger.Info().Str("action", "wait cams").Msg("lan")

	// Then ensure no camera goroutine is leaked running in the background
	lan.camsSwarm.Cancel()
	lan.camsSwarm.Wait()
}

func (lan *lanAgent) Cameras() []*LanCamera {
	lan.dataLock.Lock()
	defer lan.dataLock.Unlock()
	out := make([]*LanCamera, 0, len(lan.devices))
	for _, cam := range lan.devices {
		out = append(out, cam)
	}
	return out
}

// RunTimers runs the main loop of the agent to trigger periodical actions
func (lan *lanAgent) RunTimers(ctx context.Context) {
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

// discoverNics does the discovery from the output of a given function.
// It helps to test the logic.
func (lan *lanAgent) discoverNics() error {
	itfs, err := utils.DiscoverSystemNics()
	if err != nil {
		return errors.Trace(err)
	}

	utils.Logger.Trace().Strs("interfaces", itfs).Msg("disc")

	for _, itf := range itfs {
		lan.maybeRegisterInterface(itf)
	}

	for _, itf := range lan.interfacesStatic {
		utils.Logger.Info().Str("itf", itf).Str("action", "force").Msg("disc")
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
			utils.Logger.Warn().Str("pattern", pattern0).Str("itf", itf).Err(err).Msg("disc")
		} else if !match {
			continue
		} else if !not {
			utils.Logger.Info().Str("pattern", pattern0).Str("itf", itf).Str("action", "add").Msg("disc")
			lan.registerInterface(itf)
		} else {
			utils.Logger.Debug().Str("pattern", pattern0).Str("itf", itf).Str("action", "skip").Msg("disc")
		}
		return
	}
}

func (lan *lanAgent) registerInterface(itf string) {
	lan.interfaces.Add(NewNIC(itf))
}

func (lan *lanAgent) learnSingleCameraSync(ctx context.Context, generation uint32, discovered sdk.Appliance) error {
	dev, err := NewCamera(discovered)
	if err != nil {
		return errors.Trace(err)
	}

	lan.dataLock.Lock()
	defer lan.dataLock.Unlock()

	// If the camera is already know, let's just update its generation counter
	devInPlace, ok := lan.devices.Get(dev.PK())
	if ok {
		if generation > devInPlace.generation {
			utils.Logger.Debug().
				Str("key", devInPlace.PK()).
				Str("endpoint", discovered.GetEndpoint("device")).
				Uint32("gen", generation).
				Str("action", "refresh").
				Msg("device")
			devInPlace.generation = generation
		} // else ... stale discovery (why not?)
	} else {
		dev.generation = generation
		lan.devices.Add(dev)
		utils.Logger.Info().
			Str("key", dev.PK()).
			Str("endpoint", discovered.GetEndpoint("device")).
			Uint32("gen", generation).
			Str("action", "add").
			Msg("device")

		lan.camsSwarm.Run(runCam(dev))
		lan.Notify(dev.PK(), CameraState_Online)
	}
	return nil
}

func (lan *lanAgent) learnAllCamerasSync(ctx context.Context, gen uint32, discovered []sdk.Appliance) {
	// Update the devices that match the
	for _, dev := range discovered {
		if err := lan.learnSingleCameraSync(ctx, gen, dev); err != nil {
			utils.Logger.Warn().Str("url", dev.GetEndpoint("device")).Err(err).Msg("invalid device discovered")
		}
	}

	toBePurged := lan.camsToBePurged(gen)

	utils.Logger.Trace().Str("action", "purge").Interface("count", len(toBePurged)).Msg("lan")
	for _, dev := range toBePurged {
		lan.dataLock.Lock()
		lan.devices.Remove(dev.PK())
		lan.Notify(dev.PK(), CameraState_Offline)
		dev.StopStream()
		lan.dataLock.Unlock()
	}
}

func (lan *lanAgent) camsToBePurged(gen uint32) []*LanCamera {
	out := make([]*LanCamera, 0)

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

func (lan *lanAgent) triggerRescanAsync(ctx context.Context) {
	gen := atomic.AddUint32(&lan.generation, 1)
	utils.Logger.Trace().Str("action", "rescan").Uint32("gen", gen).Msg("lan")
	for _, itf := range lan.interfaces {
		itf.TriggerRescanAsync(ctx, gen)
	}
}

func (lan *lanAgent) PK() string { return "lan" }
