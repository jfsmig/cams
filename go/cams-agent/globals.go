package main

import "net/http"

var (
	User         = "admin"
	Password     = "ollyhgqo"
	UpstreamAddr = "127.0.0.1:6000"
)

var (
	HttpClient = http.Client{}
)
