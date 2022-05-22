// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

type TLSConfig struct {
	PathCrt string `json:"crt"`
	PathKey string `json:"key"`
}

type HubConfig struct {
	Listen string    `json:"listen"`
	Tls    TLSConfig `json:"tls"`
}
