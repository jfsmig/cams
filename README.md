# Cams / Social Video Network

Expose your cameras to a community of trust.

## Architecture

1. Streaming **devices** are present on the field, foster those implementing the [OnVif] standard protocol.
2. An **agent** is deployed on each site, close to the cameras, i.e. on the same LAN, an agent...
   * carries the credentials of the user 
   * discovers of the devices (if not relying on a static configuration)
   * pilots the local cameras.
   * registers the streams in a Hub
   * tunnels the desired stream toward the Hub
3. A **Hub** on a cloud...
   * Authenticates the users and the devices
   * Manage quotas and QoS
   * Authorizes the actions of the users toward the devices
   * Require the agents to Play/Pause media streams 
   * Efficiently Route the media streams from the devices toward the viewers

The agent:
* On the LAN side:
  * [WS Discovery], SOAP messages over Multicast UDP (239.255.255.250:8307)
  * [OnVif] protocol to control the devices and discover their media streams (HTTP port 8000, XML, SOAP)
  * [RTSP] over UDP to control the media streams
  * [RTP] and [RTCP] over UDP to consume the media streams
* Toward the Hub's Concentrator:
  * [gRPC] with both uni-directional RPC and bi-directional streaming of messages
* Toward the Hub's Streamer:
  * Emit [RTP] and [RTCP]

The Hub Concentrator
* Toward the agents:
  * Receives registrations
  * Emit commands to control the streams: `Play`, `Pause`
* Toward the Streamer

The Hub

## Installation guide

First, install the dependencies
```shell
# system deps
sudo apt install protobuf-compiler protobuf-compiler-grpc protoc-gen-go
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Golang deps
go mod download
```

If necessary, refresh the generated code
```shell
go generate ./...
```

Then build all the parts of the Cam Hub system
```shell
go install ./...
```

Then, optionally run the test suite
```shell
go test ./...
```

## References
* RFC for RTP [rfc3550]
* RFC for extesnions of RTCP [rfc5760]
* RFC for RTSP [rfc2326]
* [OnVif]
* [gRPC]

[rfc2326]: https://datatracker.ietf.org/doc/html/rfc2326
[rfc3550]: https://datatracker.ietf.org/doc/html/rfc3550
[rfc5760]: https://datatracker.ietf.org/doc/html/rfc5760
[gRPC]: https://grpc.io/
[OnVif]: https://www.onvif.org/
[RTP]: https://en.wikipedia.org/wiki/Real-time_Transport_Protocol
[RTCP]: https://en.wikipedia.org/wiki/RTP_Control_Protocol
[RTSP]: https://en.wikipedia.org/wiki/Real_Time_Streaming_Protocol
[WS Discovery]: https://en.wikipedia.org/wiki/Web_Services_Discovery
