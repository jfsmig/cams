# Architecture

Cams is a distributed system helping to share home cameras with a community of trust.

## The role of each entity

### Devices

Streaming **devices** are present on the field, foster those implementing the [OnVif]
standard protocol.

### Agent

An **agent** is deployed on each LAN, close to the cameras. It...
* carries the credentials of the user
* discovers the devices (if not relying on a static configuration)
* pilots the local cameras
* registers the streams in a Hub
* tunnels the desired streams toward the Hub

Its protocols:
* On the LAN side:
  * [WS Discovery], SOAP messages over Multicast UDP (239.255.255.250:3702)
  * [OnVif] to control the devices and discover their media streams (HTTP port 8000, XML, SOAP)
  * [RTSP] over TCP to negotiate and control the media streams
  * [RTP] and [RTCP] over UDP to consume the media streams
* Toward the Hub's control plane:
  * [gRPC], with both uni-directional RPC and bi-directional streaming of messages
* Toward the Hub's data plane:
  * emit the raw [RTP] and [RTCP] packets, encapsulated in [gRPC]

### Hub

A **Hub** runs on a cloud and splits in two planes, which are two processes:
`go/bin/cams-ctrl` on port 6000 and `cpp/hub`'s `cams-rtp2hls` on port 6001.
They share `api/hub.proto` and no state at all.

This section is the intent. What is built is marked, because the gap is large
enough to mislead.

The **control plane** (`cams-ctrl`):
* Receives the registrations of the streams from the agents — **built**
* Requires the agents to `Play` / `Pause` media streams — **built**
* Authenticates the users and the devices — **not built**: an agent says who it
  is and the hub believes it, and `utils.DialTLS` returns `NotImplemented`
* Authorizes the actions of the users toward the devices — **not built**;
  `Viewer.Play` and `Viewer.Pause` carry no credentials
* Manages quotas — **not built**
* Declares the stream expectations toward the data plane — **not built**: there
  is no link between the two planes

The **data plane** (`cams-rtp2hls`):
* Receives the streams from the agents — **built**
* Remuxes each stream into fMP4 fragments under `--root`, with a playlist a
  browser can play — **built**
* Receives the stream directives from the control plane — **not built**
* Routes each stream toward the watchers — **not as described**, and
  deliberately: the fragments are files, so `cams-hls` serves them over HTTP and
  the fan-out costs nothing per watcher. Duplicating the stream per watcher is
  what HLS exists to avoid.

### Frontend

The **frontend** would register the watchers in the control plane.

The serving half exists: `cams-hls` (`go/hlsserver`) reads the same `--root` the
data plane writes and hands the playlists, initialisation segments and fragments
to a browser, with a small player page so the path can be seen working. It is a
sibling process of `cams-rtp2hls` and shares nothing with it but the directory.

What is still missing is everything about *watchers*:

* **No authorization.** Anyone who reaches the port watches every camera and can
  list them. See `AGENTS.md`.
* **No watcher registration.** Nothing in `api/hub.proto` lets a viewer announce
  itself, so the control plane does not know a stream is being watched and
  `Viewer.Pause` from anybody stops it for everybody.
* **No event path**, so "receive the event, then watch the related stream" is
  still only the second half.

## The LAN agent architecture

The agents `cams-agent` is agent-oriented.
Multiple internal agents communicate via a message bus, implemented by [Mangos].
The messages are plain text commands.

That choice comes from the large variety of long-running tasks, that can run idenpendently but that act on each other.
Each agent becomes rather simple, mostly lock-free, and its impact on other agents is just a bunch of messages.
Thus each agent is easier to test individually.

Internal agents:
* There's one agent for the Upstream connection to the Hub.
  That agent performs the registrations to the controller.
  It receives the Play/Pause commands, These commands are forwarded to the cameras.
* There's one agent for the LAN discovery of cameras.
  It acts as the camera manager that knows which camera actually exists.
  It instanciates the cameras and their respective agents.
* There's one agent for each discovered camera.
  It is instanciated by the LAN agent.

Each agent must have a controller structure, acting as a client, owning the wire
protocol of the agent.

main() owns both the upstream and lan agents, the lan agent own the cameras agent.
When an owner exits, it must leave no background agent running.

[Mangos]: https://github.com/nanomsg/mangos
[OnVif]: https://www.onvif.org/
[WS Discovery]: https://en.wikipedia.org/wiki/Web_Services_Discovery
[RTSP]: https://en.wikipedia.org/wiki/Real_Time_Streaming_Protocol
[RTP]: https://en.wikipedia.org/wiki/Real-time_Transport_Protocol
[RTCP]: https://en.wikipedia.org/wiki/RTP_Control_Protocol
[gRPC]: https://grpc.io/
