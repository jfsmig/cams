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
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/jfsmig/cams/go/lanagent"
	"github.com/jfsmig/cams/go/upagent"
	"github.com/juju/errors"
)

const (
	DefaultScanPeriod      = 60
	DefaultRegisterPeriod  = 5
	DefaultCheckPeriod     = 10
	DefaultUpstreamTimeout = 30

	// The hub is two processes on two ports, and an agent talks to both.
	//
	// DefaultControlAddr is cams-ctrl, which carries the registrations and the
	// Play/Stop commands. DefaultMediaAddr is cams-rtp2hls, which carries the
	// RTP upload and turns it into fragments; see cpp/hub. They are not
	// interchangeable: cams-ctrl answers MediaUpload with Unimplemented on
	// purpose, so pointing the media upload at it uploads nothing at all.
	DefaultControlAddr = "127.0.0.1:6000"
	DefaultMediaAddr   = "127.0.0.1:6001"
)

type UpstreamConfig struct {
	Address string `json:"address"`
	Timeout int64  `json:"timeout"`
}

type CameraConfig struct {
	Address  string `json:"address"`
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
}

type AgentConfig struct {
	User string `json:"user"`

	// CameraUser and CameraPassword are the ONVIF credentials of the local
	// devices. They used to be package-level variables in globals.go.
	CameraUser     string `json:"camera_user"`
	CameraPassword string `json:"camera_password"`

	// GraceGenerations is how many discovery rounds a camera may be missing
	// from before it is forgotten. Zero disables the purge.
	GraceGenerations uint32 `json:"grace_generations"`

	DiscoverPatterns []string `json:"discover"`
	ScanPeriod       int64    `json:"scan_period"`
	CheckPeriod      int64    `json:"check_period"`
	RegisterPeriod   int64    `json:"register_period"`

	Interfaces []string       `json:"interfaces"`
	Cameras    []CameraConfig `json:"cameras"`

	UpstreamControl UpstreamConfig `json:"control"`
	UpstreamMedia   UpstreamConfig `json:"media"`
}

func DefaultConfig() AgentConfig {
	return AgentConfig{
		User:             "plop",
		CameraUser:       "admin",
		DiscoverPatterns: []string{"!lo", "!docker.*", ".*"},
		ScanPeriod:       DefaultScanPeriod,
		CheckPeriod:      DefaultCheckPeriod,
		RegisterPeriod:   DefaultRegisterPeriod,
		UpstreamControl:  UpstreamConfig{Address: DefaultControlAddr, Timeout: DefaultUpstreamTimeout},
		UpstreamMedia:    UpstreamConfig{Address: DefaultMediaAddr, Timeout: DefaultUpstreamTimeout},
	}
}

func (cfg *AgentConfig) LoadFile(path string) error {
	if fin, err := os.Open(path); err != nil {
		return errors.Annotate(err, "open")
	} else {
		defer fin.Close()
		if encoded, err := ioutil.ReadAll(fin); err != nil {
			return errors.Annotate(err, "read")
		} else {
			return cfg.LoadBytes(encoded)
		}
	}
}

func (cfg *AgentConfig) LoadBytes(encoded []byte) error {
	if err := json.NewDecoder(bytes.NewReader(encoded)).Decode(cfg); err != nil {
		return errors.Annotate(err, "decode")
	}
	return nil
}

func (cfg *AgentConfig) LoadString(encoded string) error {
	if err := json.NewDecoder(strings.NewReader(encoded)).Decode(cfg); err != nil {
		return errors.Annotate(err, "decode")
	}
	return nil
}

// lan projects the configuration onto what the LAN agent needs. The agent has
// its own struct, without serialisation tags, so the on-disk format stays here.
func (cfg *AgentConfig) lan() lanagent.Config {
	cameras := make([]lanagent.CameraConfig, 0, len(cfg.Cameras))
	for _, cam := range cfg.Cameras {
		cameras = append(cameras, lanagent.CameraConfig{
			Address:  cam.Address,
			User:     cam.User,
			Password: cam.Password,
		})
	}

	return lanagent.Config{
		User:             cfg.CameraUser,
		Password:         cfg.CameraPassword,
		Interfaces:       cfg.Interfaces,
		DiscoverPatterns: cfg.DiscoverPatterns,
		Cameras:          cameras,
		ScanPeriod:       cfg.GetScanPeriod(),
		CheckPeriod:      cfg.GetCheckPeriod(),
		GraceGenerations: cfg.GraceGenerations,
	}
}

// upstream projects the configuration onto what the upstream agent needs.
func (cfg *AgentConfig) upstream() upagent.Config {
	return upagent.Config{
		User:           cfg.User,
		ControlAddress: cfg.UpstreamControl.Address,
		RegisterPeriod: time.Duration(cfg.RegisterPeriod) * time.Second,
		RequestTimeout: time.Duration(cfg.UpstreamControl.Timeout) * time.Second,
	}
}

func (cfg *AgentConfig) GetScanPeriod() time.Duration {
	return time.Duration(cfg.ScanPeriod) * time.Second
}

func (cfg *AgentConfig) GetCheckPeriod() time.Duration {
	return time.Duration(cfg.CheckPeriod) * time.Second
}
