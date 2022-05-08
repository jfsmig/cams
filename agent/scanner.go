package main

import (
	"context"
	goonvif "github.com/use-go/onvif"
	"sync"
)

type LanScanner struct {
	ItfName string
	trigger chan uint32
}

func NewLanScanner(name string) *LanScanner {
	return &LanScanner{
		ItfName: name,
		trigger: make(chan uint32, 8),
	}
}

type RegistrationFunc func(ctx context.Context, gen uint32, discovered []goonvif.Device)

func (ls *LanScanner) RunLoop(ctx context.Context, wg *sync.WaitGroup, register RegistrationFunc) {
	defer wg.Done()
	Logger.Info().Str("name", ls.ItfName).Str("action", "run").Msg("interface")
	for {
		select {
		case <-ctx.Done():
			Logger.Info().Str("name", ls.ItfName).Str("action", "stop").Msg("interface")
			close(ls.trigger)
			return
		case generation := <-ls.trigger:
			Logger.Info().Str("name", ls.ItfName).Str("action", "discover").Msg("interface")
			devices := goonvif.GetAvailableDevicesAtSpecificEthernetInterface(ls.ItfName)
			register(ctx, generation, devices)
		}
	}
}

func (ls *LanScanner) RescanAsync(ctx context.Context, generation uint32) {
	select {
	case <-ctx.Done():
		// generate no trigger if waiting for an exit
	case ls.trigger <- generation:
		// try to write a trigger token but ...
	default:
		// ... never wait if there are already pending trigger tokens
	}
}
