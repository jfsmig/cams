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
	"testing"
)

var encoded = `{
    "discover": ["eth*", "enp*", "!wl*", "!lo"],
    "scan_period": 5,
    "check_period": 5,
    "register_period": 5,
    "interfaces": [ "eno0" ],
    "cameras": [
        {"address": "127.0.0.1", "user":"admin" },
        {"address": "127.0.0.2", "user":"admin" }
    ],
    "control": {"address": "127.0.0.1:6000", "timeout": 10},
    "media": {"address": "127.0.0.1:6000", "timeout": 10}
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
	assertValue(t, decoded.UpstreamControl, expected.UpstreamControl)
	assertValue(t, decoded.UpstreamMedia, expected.UpstreamMedia)
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
		RegisterPeriod:   5,
		Interfaces:       []string{"eno0"},
		Cameras: []CameraConfig{
			{Address: "127.0.0.1", User: "admin"},
			{Address: "127.0.0.2", User: "admin"},
		},
		UpstreamControl: UpstreamConfig{Address: "127.0.0.1:6000", Timeout: 10},
		UpstreamMedia:   UpstreamConfig{Address: "127.0.0.1:6000", Timeout: 10},
	})
}

// TestConfig_Override validates that the arrays are not appended when values
// are present in the encoded form
func TestConfig_Override(t *testing.T) {
	cfg := AgentConfig{
		DiscoverPatterns: []string{"x", "y"},
		ScanPeriod:       17,
		CheckPeriod:      18,
		RegisterPeriod:   19,
		Interfaces:       []string{"xxxx"},
		Cameras: []CameraConfig{
			{Address: "127.0.0.1", User: "x"},
			{Address: "127.0.0.2", User: "y"},
			{Address: "127.0.0.2", User: "Z"},
		},
		UpstreamControl: UpstreamConfig{Address: "127.0.0.1:6001", Timeout: 10},
		UpstreamMedia:   UpstreamConfig{Address: "127.0.0.1:6001", Timeout: 10},
	}
	if err := cfg.LoadString(encoded); err != nil {
		t.Fatal(err)
	}

	assertConfig(t, cfg, AgentConfig{
		DiscoverPatterns: []string{"eth*", "enp*", "!wl*", "!lo"},
		ScanPeriod:       5,
		CheckPeriod:      5,
		RegisterPeriod:   5,
		Interfaces:       []string{"eno0"},
		Cameras: []CameraConfig{
			{Address: "127.0.0.1", User: "admin"},
			{Address: "127.0.0.2", User: "admin"},
		},
		UpstreamControl: UpstreamConfig{Address: "127.0.0.1:6000", Timeout: 10},
		UpstreamMedia:   UpstreamConfig{Address: "127.0.0.1:6000", Timeout: 10},
	})
}
