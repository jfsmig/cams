// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main


import (
	"context"
	"os"
	"sync"
	"time"
    "encoding/json"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

"""

discover wl*
scan 30s
cam 127.0.0.2
uplink 127.0.0.3:6000
"""

type UpstreamConfig struct {
	Address string        `json:"address"`
	Timeout time.Duration `json:"timeout"`
}

type CameraConfig struct {
	Address  string `json:"address"`
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
}

type InterfaceConfig struct {
	Name   string `json:"name"`
}

type AgentConfig struct {
    DiscoverPatterns []string      `json:"discover"`
	ScanPeriod       time.Duration `json:"scan_period"`

	Interfaces []InterfaceConfig `json:"interfaces"`
	Cameras    []CameraConfig    `json:"cameras"`
	Upstreams  []UpstreamConfig  `json:"upstreams"`
}

