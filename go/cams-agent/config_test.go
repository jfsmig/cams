// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"testing"
)

var encoded = `{
    "discover": ["eth*", "enp*", "!wl*", "!lo"],
    "scan_period": 5,
    "check_period": 5,
    "interfaces": [ "eno0" ],
    "cameras": [
        {"address": "127.0.0.1", "user":"admin" },
        {"address": "127.0.0.2", "user":"admin" }
    ],
    "upstreams": {"address": "127.0.0.1:6000", "timeout": 10}
}`

func assertValue[T comparable](t *testing.T, decoded, expected T) {
	if expected != decoded {
		t.Fatal("diff", expected, decoded)
	}
}

func assertArrays[T comparable](t *testing.T, decoded, expected []T) {
	if len(expected) != len(decoded) {
		t.Fatal(len(expected), len(decoded))
	}
	for i := 0; i < len(expected); i++ {
		assertValue(t, expected[i], decoded[i])
	}
}

func assertConfig(t *testing.T, decoded, expected AgentConfig) {
	assertValue(t, decoded.ScanPeriod, expected.ScanPeriod)
	assertValue(t, decoded.CheckPeriod, expected.CheckPeriod)
	assertArrays(t, decoded.DiscoverPatterns, expected.DiscoverPatterns)
	assertValue(t, decoded.Upstream, expected.Upstream)
	assertArrays(t, decoded.Interfaces, expected.Interfaces)
	assertArrays(t, decoded.Cameras, expected.Cameras)
}

func TestConfig_FromEmpty(t *testing.T) {
	var cfg AgentConfig
	if err := cfg.LoadString(encoded); err != nil {
		t.Fatal(err)
	}
	assertConfig(t, cfg, AgentConfig{
		DiscoverPatterns: []string{"eth*", "enp*", "!wl*", "!lo"},
		ScanPeriod:       5,
		CheckPeriod:      5,
		Interfaces:       []string{"eno0"},
		Cameras: []CameraConfig{
			{Address: "127.0.0.1", User: "admin"},
			{Address: "127.0.0.2", User: "admin"},
		},
		Upstream: UpstreamConfig{Address: "127.0.0.1:6000", Timeout: 10},
	})
}

// TestConfig_Override validates that the arrays are not appended when values
// are present in the encoded form
func TestConfig_Override(t *testing.T) {
	cfg := AgentConfig{
		DiscoverPatterns: []string{"x", "y"},
		ScanPeriod:       17,
		CheckPeriod:      18,
		Interfaces:       []string{"xxxx"},
		Cameras: []CameraConfig{
			{Address: "127.0.0.1", User: "x"},
			{Address: "127.0.0.2", User: "y"},
			{Address: "127.0.0.2", User: "Z"},
		},
		Upstream: UpstreamConfig{Address: "127.0.0.1:6001", Timeout: 10},
	}
	if err := cfg.LoadString(encoded); err != nil {
		t.Fatal(err)
	}

	assertConfig(t, cfg, AgentConfig{
		DiscoverPatterns: []string{"eth*", "enp*", "!wl*", "!lo"},
		ScanPeriod:       5,
		CheckPeriod:      5,
		Interfaces:       []string{"eno0"},
		Cameras: []CameraConfig{
			{Address: "127.0.0.1", User: "admin"},
			{Address: "127.0.0.2", User: "admin"},
		},
		Upstream: UpstreamConfig{Address: "127.0.0.1:6000", Timeout: 10},
	})
}
