# WiY  / Watch It Yourself

1. Streaming **devices**, i.e. IP cameras that implement the [OnVif](https://www.onvif.org/) protocol.
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

## Architecture

The agent:
* On the LAN side
  * **Web Service Discovery** of the devices: SOAP messages over Multicast UDP (239.255.255.250:8307)
  * **OnVif** protocol to control the devices and discover their media streams
  * **RTSP** over UDP to control the media streams
  * **RTP/RTSP** over UDP to consume the media streams
* Toward the Hub
  * **gRPC** with bi-directional streaming of messages

The Hub
* Receives registrations

## Installation guide

```shell
go mod download
sudo apt install protobuf-compiler protobuf-compiler-grpc protoc-gen-go
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
protoc --go_out=proto --go_opt=paths=source_relative --go-grpc_out=proto --go-grpc_opt=paths=source_relative wiy.proto
```

## References

