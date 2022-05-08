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

	"github.com/juju/errors"
)

const (
	DefaultScanPeriod      = 60
	DefaultRegisterPeriod  = 5
	DefaultCheckPeriod     = 10
	DefaultUpstreamTimeout = 30
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
		DiscoverPatterns: []string{"!lo", "!docker.*", ".*"},
		ScanPeriod:       DefaultScanPeriod,
		CheckPeriod:      DefaultCheckPeriod,
		RegisterPeriod:   DefaultRegisterPeriod,
		UpstreamControl:  UpstreamConfig{Address: "127.0.0.1:6000", Timeout: DefaultUpstreamTimeout},
		UpstreamMedia:    UpstreamConfig{Address: "127.0.0.1:6000", Timeout: DefaultUpstreamTimeout},
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

func (cfg *AgentConfig) GetScanPeriod() time.Duration {
	return time.Duration(cfg.ScanPeriod) * time.Second
}

func (cfg *AgentConfig) GetCheckPeriod() time.Duration {
	return time.Duration(cfg.CheckPeriod) * time.Second
}
