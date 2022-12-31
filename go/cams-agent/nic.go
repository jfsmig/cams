// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"github.com/jfsmig/onvif/networking"

	wsdiscovery "github.com/jfsmig/onvif/ws-discovery"

	"github.com/jfsmig/cams/go/utils"
)

type Nic struct {
	ItfName string
	trigger chan uint32
}

func NewNIC(name string) *Nic {
	return &Nic{
		ItfName: name,
		trigger: make(chan uint32, 1),
	}
}

func (ls *Nic) PK() string { return ls.ItfName }

type RegistrationFunc func(ctx context.Context, gen uint32, discovered []networking.ClientInfo)

func (ls *Nic) RunRescanLoop(ctx context.Context, register RegistrationFunc) {
	utils.Logger.Debug().Str("itf", ls.ItfName).Str("action", "start").Msg("nic")
	for {
		select {

		case <-ctx.Done():
			utils.Logger.Info().Str("itf", ls.ItfName).Str("action", "stop").Msg("nic")
			close(ls.trigger)
			return

		case generation := <-ls.trigger:
			devices, err := wsdiscovery.GetAvailableDevicesAtSpecificEthernetInterface(ls.ItfName)
			if err != nil {
				utils.Logger.Warn().Str("action", "rescan").Str("itf", ls.ItfName).Uint32("gen", generation).Err(err).Msg("nic")
				continue
			}

			utils.Logger.Trace().Str("action", "rescan").Int("devices", len(devices)).Str("itf", ls.ItfName).Uint32("gen", generation).Msg("nic")
			register(ctx, generation, devices)
		}
	}
}

func (ls *Nic) TriggerRescanAsync(ctx context.Context, generation uint32) {
	select {
	case <-ctx.Done():
		// generate no trigger if waiting for an exit
	case ls.trigger <- generation:
		// try to write a trigger token but ...
		utils.Logger.Info().Str("action", "rescan triggered").Str("itf", ls.ItfName).Uint32("gen", generation).Msg("nic")
	default:
		utils.Logger.Warn().Str("action", "rescan avoided").Str("itf", ls.ItfName).Uint32("gen", generation).Msg("nic")
		// ... never wait if there are already pending trigger tokens
	}
}
