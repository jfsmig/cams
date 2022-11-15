// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"net"
	"net/url"
	"regexp"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jfsmig/go-bags"
	"github.com/juju/errors"
	goonvif "github.com/use-go/onvif"

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
		defer lan.dataLock.Lock()
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
			lan.camsSwarm.Run(cam.Run)
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
	// This is not the case for network interfaces that are rather stable.
	lan.nicsGroup = utils.NewGroup(ctx)
	lan.camsSwarm = utils.NewSwarm(ctx)

	// Perform a first discovery of the local interfaces.
	// No need to do it periodically, interfaces are unlikely plug & play
	if err := lan.discoverNics(); err != nil {
		utils.Logger.Error().Err(err).Msg("discovery")
		return
	}

	// Spawn one goroutine per registered interface, for concurrent discoveries
	fn := func(ctx0 context.Context, gen uint32, devs []goonvif.Device) {
		lan.learnAllCamerasSync(ctx0, gen, devs)
	}
	for _, itf := range lan.interfaces {
		lan.nicsGroup.Run(func(c context.Context) { itf.RunRescanLoop(c, fn) })
	}

	lan.nicsGroup.Run(func(c context.Context) { lan.RunTimers(c) })

	// Wait for the discovery goroutines to stop, this will happen until a strong
	// error condition occurs.
	lan.nicsGroup.Wait()

	// Then ensure no camera goroutine is leaked running in the background
	lan.camsSwarm.Cancel()
	lan.camsSwarm.Wait()
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

func (lan *lanAgent) learnSingleCameraSync(ctx context.Context, generation uint32, discovered goonvif.Device) error {
	k := discovered.GetDeviceInfo().SerialNumber

	u := discovered.GetEndpoint("device")
	utils.Logger.Debug().Str("url", u).Uint32("gen", generation).Str("action", "adding").Msg("device")

	parsedUrl, err := url.Parse(u)
	if err != nil {
		return errors.Trace(err)
	}

	lan.dataLock.Lock()
	defer lan.dataLock.Lock()

	// If the camera is already know, let's just update its generation counter
	devInPlace, ok := lan.devices.Get(k)
	if ok {
		if generation > devInPlace.generation {
			devInPlace.generation = generation
		}
		return nil
	}

	// If not, the camera is new on the LAN
	dev, err := NewCamera(k, parsedUrl.Host)
	if err == nil {
		dev.generation = generation
		lan.devices.Add(dev)
		utils.Logger.Info().
			Str("key", dev.PK()).
			Str("endpoint", u).
			Str("action", "add").
			Str("user", dev.user).
			Str("password", dev.password).
			Msg("device")
		lan.Notify(dev.PK(), CameraState_Online)
	}
	return err
}

func (lan *lanAgent) learnAllCamerasSync(ctx context.Context, gen uint32, discovered []goonvif.Device) {
	// Update the devices that match the
	for _, dev := range discovered {
		if err := lan.learnSingleCameraSync(ctx, gen, dev); err != nil {
			utils.Logger.Warn().Str("url", dev.GetEndpoint("device")).Err(err).Msg("invalid device discovered")
		}
	}

	lan.dataLock.Lock()
	defer lan.dataLock.Unlock()

	// Unregister and shut the devices from older generations
	for i := len(lan.devices); i > 0; i-- {
		dev := lan.devices[i-1]
		if dev.generation < (gen - lan.GraceGenerations) {
			lan.devices.Remove(dev.PK())
			lan.Notify(dev.PK(), CameraState_Offline)
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

func (lan *lanAgent) PK() string { return "lan" }

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
