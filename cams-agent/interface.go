// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"github.com/jfsmig/cams/utils"
	goonvif "github.com/use-go/onvif"
)

type LanInterface struct {
	ItfName string
	trigger chan uint32
}

func NewLanInterface(name string) *LanInterface {
	return &LanInterface{
		ItfName: name,
		trigger: make(chan uint32, 8),
	}
}

func (ls *LanInterface) PK() string { return ls.ItfName }

type RegistrationFunc func(ctx context.Context, gen uint32, discovered []goonvif.Device)

func (ls *LanInterface) RunRescanLoop(ctx context.Context, register RegistrationFunc) {
	utils.Logger.Debug().Str("name", ls.ItfName).Str("action", "run").Msg("interface")
	for {
		select {
		case <-ctx.Done():
			utils.Logger.Info().Str("name", ls.ItfName).Str("action", "stop").Msg("interface")
			close(ls.trigger)
			return
		case generation := <-ls.trigger:
			devices, err := goonvif.GetAvailableDevicesAtSpecificEthernetInterface(ls.ItfName)
			if err == nil {
				register(ctx, generation, devices)
			} else {
				utils.Logger.Warn().Str("action", "rescan").Err(err).Msg("interface")
			}
		}
	}
}

func (ls *LanInterface) TriggerRescanAsync(ctx context.Context, generation uint32) {
	select {
	case <-ctx.Done():
		// generate no trigger if waiting for an exit
	case ls.trigger <- generation:
		// try to write a trigger token but ...
	default:
		// ... never wait if there are already pending trigger tokens
	}
}