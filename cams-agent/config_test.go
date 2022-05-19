// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
    "testing"
)

func TestConfig0(t *testing.T) {
    encoded := '''{
    "discover": ["eth*", "enp*", "!wl*"],
    "scan_period": "5s",
    "interfaces": [ "eno0" ],
    "cameras": [
        {"address": "127.0.0.1", "user":"admin" },
        {"address": "127.0.0.2", "user":"admin" }
    ],
    "upstreams": [
        {"address": "127.0.0.1:6000", "timeout": "10s"}
    ]
}'''

    cfg := AgentConfig{}
    
}

