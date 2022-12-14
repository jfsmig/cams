// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"fmt"
	"net/url"

	"github.com/juju/errors"
	goonvif "github.com/use-go/onvif"
	"github.com/use-go/onvif/device"
	sdk "github.com/use-go/onvif/sdk/device"

	"github.com/jfsmig/cams/go/utils"
)

func discover(ctx context.Context) error {
	interfaces, err := utils.DiscoverSystemNics()
	if err != nil {
		return errors.Annotate(err, "lan discovery")
	}

	for _, itf := range interfaces {
		devices, err := goonvif.GetAvailableDevicesAtSpecificEthernetInterface(itf)
		if err != nil {
			utils.Logger.Warn().Str("itf", itf).Msg("lan discovery failed")
		} else {
			for _, dev := range devices {
				u := dev.GetEndpoint("device")
				parsedUrl, err := url.Parse(u)
				authDev, err := goonvif.NewDevice(goonvif.DeviceParams{
					Xaddr:    parsedUrl.Host,
					Username: "admin",
					Password: "ollyhgqo",
				})
				if err != nil {
					utils.Logger.Warn().Str("itf", itf).Msg("auth failed")
				} else {
					dev = *authDev
				}
				reply, err := sdk.Call_GetDeviceInformation(ctx, &dev, device.GetDeviceInformation{})
				fmt.Println(itf, dev.GetDeviceInfo(), dev.GetServices(), reply, err)
			}
		}
	}
	return nil
}
