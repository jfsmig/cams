# Cams / Social Video Network

Expose your cameras to a community of trust.

See [ARCHITECTURE.md](ARCHITECTURE.md) for the entities of the system and the protocols
between them, and [AGENTS.md](AGENTS.md) for the engineering rules and the build.


## Installation guide

First, install the dependencies
```shell
# system deps: protoc itself, plus the C++ gRPC plugin used by cpp/hub
sudo apt install protobuf-compiler protobuf-compiler-grpc

# protoc plugins for Go, pinned to the versions that produced the committed bindings
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.32.0
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

# Golang deps
cd go
go mod download
```

Those plugin versions are not arbitrary. They must move together with the
`google.golang.org/protobuf` and `google.golang.org/grpc` runtimes pinned in `go/go.mod`;
installing them at `@latest` emits code the pinned runtimes cannot compile. Read the
warning in [AGENTS.md](AGENTS.md) before bumping either side.

If necessary, refresh the generated code
```shell
cd go
go generate ./api/pb/...
```

Then build all the parts of the Cam Hub system
```shell
cd go
go install ./...
```

Then, optionally run the test suite
```shell
cd go
go test ./...
go test -race ./...
```

The C++ hub builds separately, with CMake, configured from `cpp/`:
```shell
cmake -S cpp -B cpp/build -DCMAKE_BUILD_TYPE=Debug
cmake --build cpp/build --parallel
ctest --test-dir cpp/build --output-on-failure
```

It turns the agents' RTP upload into HLS over fragmented MP4, using libav* for both the
RFC 6184 depacketization and the muxing, and needs FFmpeg 7 or newer. It **remuxes** and
never decodes: the camera already emits H.264, so reaching HLS is a container change
whose cost does not depend on how many people watch. Its `ctest` replays a recorded
camera stream end to end, so it exercises the media path without a camera.

### Watching a stream

`cams-hls` serves what the hub wrote, and carries a small player page so the
media path can be seen rather than only tested:

```shell
cams-rtp2hls --root /tmp/hls          # the data plane, on :6001
cams-hls     --root /tmp/hls          # the viewer side, on :6002
```

Then open <http://127.0.0.1:6002/> and pick a camera. The page prefers the
browser's native HLS, which is what Safari has and the only path that plays
H.265, and falls back to hls.js elsewhere. It reports how long it took to get a
picture, and offers a way back to the live edge of the retained window.

**`cams-hls` has no authorization.** Anyone who can reach its port can watch
every camera on the hub and enumerate them, which is why it listens on loopback
by default. Do not expose it without putting something that authenticates in
front of it.

Everything the CI systems check is in `ci/`; `ci/all.sh` runs all of it.

## References
* RFC for RTSP [rfc2326]
* RFC for RTP [rfc3550]
* RFC for extensions of RTCP [rfc5760]
* [OnVif]
* [gRPC]


## Golang RTSP / RTP / RTCP

If you are interested in a very good library to handle an IP camera, consider
[gortsplib]. [Aler9] wrote it and he is to be praised: it is very good, it moves fast,
and it handles the weirdness of heterogeneous hardware. Cams depends on it.

It used to be otherwise. `go/rtsp1` held a modified copy of an early `aler9/gortsplib`,
forked to cut the library at the boundary between the signalling and the data plane. The
need was narrow: cams captures the raw RTP and RTCP packets and tunnels them to a hub
that performs the decoding, so the agent wants a hand on every single packet and no
heavyweight parsing on the field. With the library of the time that turned out to be very
complicated, and the fork went as far as deleting the whole data plane so that the agent
could own its own sockets.

Upstream has since made that fork pointless. `Client.ListenPacket` lets the caller create
the UDP sockets, `Setup` still accepts explicit client ports, `DialContext` accepts any
`net.Conn`, and RTSP carried over another transport is first-class through `Tunnel`. Cams
now depends on [gortsplib] v5, `go/rtsp1` is gone, and `go/camagent` keeps its hand on
every packet through the `OnPacketRTPAny` and `OnPacketRTCPAny` callbacks — which fire
identically whether the media arrives over UDP or over interleaved TCP.


[aler9]: https://github.com/aler9
[gortsplib]: https://github.com/bluenviron/gortsplib

[rfc2326]: https://datatracker.ietf.org/doc/html/rfc2326
[rfc3550]: https://datatracker.ietf.org/doc/html/rfc3550
[rfc5760]: https://datatracker.ietf.org/doc/html/rfc5760
[gRPC]: https://grpc.io/
[OnVif]: https://www.onvif.org/
