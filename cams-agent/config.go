// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"bytes"
	"encoding/json"
	"github.com/juju/errors"
	"io/ioutil"
	"os"
	"strings"
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

	Interfaces []string       `json:"interfaces"`
	Cameras    []CameraConfig `json:"cameras"`
	Upstream   UpstreamConfig `json:"upstreams"`
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
