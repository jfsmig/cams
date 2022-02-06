// Code generated : DO NOT EDIT.

// Copyright (c) 2022 Jean-Francois Smigielski
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"encoding/xml"
	"io/ioutil"

	goonvif "github.com/use-go/onvif"
	"github.com/use-go/onvif/media"
)

func call_GetStreamUri_parse_GetStreamUriResponse(dev goonvif.Device, request media.GetStreamUri) (media.GetStreamUriResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetStreamUriResponse media.GetStreamUriResponse
		}
	}

	var reply Envelope

	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.GetStreamUriResponse, err
	} else {
		// FIXME(jfs): Get rid of this buffering
		if b, err := ioutil.ReadAll(httpReply.Body); err != nil {
			return reply.Body.GetStreamUriResponse, err
		} else {
			if err = xml.Unmarshal(b, &reply); err != nil {
				return reply.Body.GetStreamUriResponse, err
			} else {
				return reply.Body.GetStreamUriResponse, nil
			}
		}
	}
}

