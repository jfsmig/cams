// Copyright (c) 2022-2024 The authors (see the AUTHORS file)
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"context"

	"github.com/jfsmig/cams/go/utils"
	"github.com/jfsmig/onvif/networking"
	wsdiscovery "github.com/jfsmig/onvif/ws-discovery"
	"github.com/rs/zerolog"
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

func (ls *Nic) debug() *zerolog.Event { return utils.Logger.Debug().Str("itf", ls.PK()) }

func (ls *Nic) warn(err error) *zerolog.Event {
	return utils.Logger.Warn().Err(err).Str("itf", ls.PK())
}

type RegistrationFunc func(ctx context.Context, gen uint32, discovered []networking.ClientInfo)

func (ls *Nic) RunRescanLoop(ctx context.Context, register RegistrationFunc) {
	ls.debug().Msg("nic starting")
	for {
		select {

		case <-ctx.Done():
			ls.debug().Msg("nic stopping")
			close(ls.trigger)
			return

		case generation := <-ls.trigger:
			devices, err := wsdiscovery.GetAvailableDevicesAtSpecificEthernetInterface(ls.ItfName)
			if err != nil {
				ls.warn(err).Uint32("gen", generation).Msg("nic rescan failure")
				continue
			}
			if len(devices) > 0 {
				ls.debug().Uint32("gen", generation).Int("found", len(devices)).Msg("nic rescan success")
			}
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
	default:
		ls.debug().Uint32("gen", generation).Msg("nic rescan avoided")
		// ... never wait if there are already pending trigger tokens
	}
}
