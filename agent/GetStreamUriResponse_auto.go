// Code generated : DO NOT EDIT.

// Copyright (c) 2022 Jean-Francois Smigielski
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"encoding/xml"
	"io/ioutil"
	"github.com/juju/errors"
	goonvif "github.com/use-go/onvif"
	"github.com/use-go/onvif/media"
)

func call_GetStreamUri_parse_GetStreamUriResponse(dev *goonvif.Device, request media.GetStreamUri) (media.GetStreamUriResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetStreamUriResponse media.GetStreamUriResponse
		}
	}

	var reply Envelope

	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.GetStreamUriResponse, errors.Trace(err)
	} else {
		Logger.Debug().
			Str("msg", httpReply.Status).
			Int("status", httpReply.StatusCode).
			Str("rpc", "GetStreamUriResponse").
			Msg("RPC")

		// FIXME(jfs): Get rid of this buffering
		b, err := ioutil.ReadAll(httpReply.Body)
		if err != nil {
			return reply.Body.GetStreamUriResponse, errors.Trace(err)
		}
		if err = xml.Unmarshal(b, &reply); err != nil {
			return reply.Body.GetStreamUriResponse, errors.Trace(err)
		} 
		return reply.Body.GetStreamUriResponse, nil
	}
}

