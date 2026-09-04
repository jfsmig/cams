# Cams

## What it is

first read [README.md](README.md). It explains the purpose of the project.

Then read [ARCHITECTURE.md](ARCHITECTURE.md). It explains the software entities.

The whole project is licensed under the **AGPL-3.0**. Every source file carries the
header; keep it on new files.

## Layout

| Path | Contents |
|---|---|
| `api/hub.proto` | The gRPC contract between all the components. **Source of truth.** |
| `go/` | Go module `github.com/jfsmig/cams/go`. |
| `go/api/pb/` | Go bindings generated from `api/hub.proto`. Never hand-edited. |
| `go/bin/` | The `cams-agent`, `cams-cli`, `cams-ctrl` and `cams-hls` entry points. |
| `go/agentbus/` | The control bus: REQ/REP, one line of text per command. Framing, client side, serving loop. No vocabulary of its own. |
| `go/mediabus/` | The data bus: PUSH/PULL, binary media frames. The camera connects, the upstream binds. |
| `go/camagent/` | The agent of one camera: ONVIF discovery, the RTSP session, and pushing the media frames onto the data bus. Imported only by `main`. |
| `go/camctrl/` | The controller of a camera agent, and the vocabulary they exchange. |
| `go/lanagent/` | The agent managing the LAN: NIC discovery, the camera registry, the camera lifecycles. Imported only by `main`. |
| `go/lanctrl/` | The controller of the LAN agent, and the vocabulary they exchange. |
| `go/upagent/` | The agent holding the link to the hub: registrations out, Play/Pause in. Imported only by `main`. |
| `go/upctrl/` | The controller of the upstream agent: `PING` and `STATE`, nothing more. |
| `go/hlsserver/` | The viewer side: serves what `cpp/hub` wrote, plus a player page to prove it. **No authorization** -- see below. |
| `go/transport/` | A raw UDP socket pair. Unused since the move to gortsplib v5. |
| `go/utils/` | Logging, gRPC dialing, LAN helpers, and the `Swarm` goroutine groups. |
| `cpp/hub/` | The hub's media plane: RTP in over gRPC, fMP4 fragments out. A chain of `PacketStage`s; see `RtpSource`, `KeyframeGate`, `DurationFiller` and `FragmentSink`. |
| `ci/` | The checks both CI systems run. `ci/go.sh`, `ci/cpp.sh`, `ci/all.sh`. |
| `docker/ci/` | The CI image, pinning the toolchain versions. |

RTSP is handled by upstream [gortsplib](https://github.com/bluenviron/gortsplib) v5. A
modified copy of it used to live in `go/rtsp1`; do not reintroduce a fork, the hooks that
motivated it (`ListenPacket`, `DialContext`, `Tunnel`) are all upstream now.

## `api/hub.proto` is the source of truth

Both ends of every wire are generated from that single file: the Go bindings under
`go/api/pb/`, which are committed, and the C++ ones, which are not -- CMake generates
them into the build tree, because the output of `protoc-gen-grpc-cpp` is tied to the
installed gRPC and a committed copy is guaranteed to disagree with somebody's toolchain.
The schema is the only place where the two languages meet, which makes it the most
expensive file in the repository to get wrong.

1. **A protocol change starts in `api/hub.proto`, never in generated code.** Editing
   `go/api/pb/*.pb.go` by hand is always wrong: the next generation silently reverts it,
   and the two languages drift apart.
2. **Regenerate both languages in the same change.** A schema touched on one side only
   still compiles on both sides and fails at runtime, which is the most expensive class
   of mistake here.
3. **Field numbers are the contract, field names are not.** Renaming a field costs
   nothing on the wire; renumbering one, or recycling a retired number, breaks every
   deployed agent. `reserved` what you retire.
4. **Agents and hub are upgraded separately.** Assume a version skew of at least one
   release in both directions, and add fields rather than repurposing them.

### Regenerating the Go bindings

From `go/api/pb/`:

```
protoc --proto_path=../../../api \
       --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       hub.proto
```

That is also what `go generate ./api/pb/...` runs, through the directive in
`go/api/pb/doc.go`.

One trap, verified:

* **The generator and the runtime must move together.** The committed bindings come from
  `protoc-gen-go` v1.32.0, matching the pinned `google.golang.org/protobuf` v1.32.0 and
  `google.golang.org/grpc` v1.61.1. A newer `protoc-gen-go-grpc` emits code that needs
  `grpc.SupportPackageIsVersion9` and the generic stream types, none of which exist in
  v1.61.1. Regenerating with newer plugins therefore requires bumping the runtimes in
  `go/go.mod` in the same change, or the tree stops compiling.

## `cams-hls` has no authorization

`go/hlsserver` serves every playlist, initialisation segment and fragment under
its `--root` to anyone who can reach its port, and `/streams` enumerates the
cameras for them. That is the whole security model, and it is why the listener
defaults to `127.0.0.1:6002` and must never default to `0.0.0.0`.

Two consequences worth keeping in mind when touching that package:

* **Path validation is the only boundary there is.** `safeStreamID` in
  `go/hlsserver/names.go` mirrors `stream_id_is_safe` in
  `cpp/hub/StreamStorage.cpp` and the two have to stay identical, because one
  names those directories and the other reads them. With no credential in
  front, a traversal bug is an unauthenticated arbitrary-file read rather than a
  privilege escalation.
* **What may be served is an allowlist, not a filter**: the playlist, the
  initialisation segment, and `seg_` + digits + `.m4s`. The directory also holds
  `stream.params`, which is the hub's own fingerprint of what it is recording,
  and the `.tmp` files the muxer renames from. Naming what may leave keeps both
  unreachable without having to enumerate them, and keeps whatever is added next
  unreachable by default.

When a credential does arrive it has to be a **cookie**, not a header and not a
query parameter. Safari's native HLS player cannot set request headers, and it
is the only browser that plays the H.265 the hub carries; and a query token does
not survive the playlist, because a player resolves `seg_00000.m4s` relative to
the playlist URL and drops the query string -- which would force rewriting every
URI and turn a reader into a transformer. `/streams` is the first thing to gate.

## Three operations, three names

The media path does one of three things to a stream, and conflating them is
expensive enough to be worth the vocabulary:

| Term | What it does | Cost per 1080p stream |
|---|---|---|
| **depacketize** | RTP payloads to access units, RFC 6184. libavformat's `sdp` demuxer, in `RtpSource`. | negligible |
| **remux** / **fragment** | Access units into fMP4 fragments. The codec is untouched: `avcodec_parameters_copy` and `av_interleaved_write_frame`, in `FragmentSink`. | ~1-2% of a core |
| **transcode** | Decode and re-encode. Not built. | ~0.5-1 core for video, ~1% for audio |

**The default path remuxes. Do not call it transcoding.** Naming it that is what
invites somebody to build the fifty-times-costlier version of a job that is
already done: an ONVIF camera emits H.264 already, so reaching HLS is a
container change and its cost does not depend on how many people watch. That is
why `cpp/hub/` contains no `avcodec_send_packet`, `AVFrame` or bitstream filter.

Nothing mechanical enforces that. `docker/ci/Dockerfile` asks only for
`libavformat-dev`, `libavcodec-dev` and `libavutil-dev`, but libavfilter,
libswscale and libswresample arrive in the image anyway as transitive
dependencies, headers and `.pc` files included, so transcoding code would
compile and link there. The boundary is held by review, not by the build.

**H.265 is remuxed too, not transcoded.** The mov muxer writes `hvcC` from the
Annex-B parameter sets exactly as it writes `avcC`, and every stage above the
sink is codec-agnostic, so the same chain carries it. What differs is
downstream: HEVC in fMP4 plays in Safari and in a Chrome with hardware decode,
and in little else, which is why the agent prefers a camera's H.264 profile
whenever it offers one -- see `chooseProfile`. The H.265 arm carries the cameras
that offer nothing else.

Transcoding therefore earns its place in fewer cases than it looks, none of
which is built: MJPEG to H.264, G.711/G.726 to AAC, and H.265 to H.264 only to
reach a browser that cannot play HEVC -- not to carry it at all. Fragment length
is **not** one of them: a fragment can only break on a keyframe, so its length
is a multiple of the camera's GOP, and the lever for that is the camera's own
encoder configuration over ONVIF, not a two-core re-encode.

## The agents talk over the bus, not through each other

`ARCHITECTURE.md` describes `cams-agent` as a set of internal agents exchanging plain
text commands over a [Mangos](https://github.com/nanomsg/mangos) bus. All three are
converted -- camera, LAN, upstream. The pattern:

1. **An agent package is imported by `main` and nothing else.** Everything else holds a
   controller. `main` builds each pair and injects a factory, which is why
   `lanagent.New` takes a `CameraFactory` rather than knowing how a camera is made. An
   agent may name a *controller* package -- `lanagent` names `camctrl`, `upagent` names
   `lanctrl` -- because a controller drags nothing behind it. Naming another *agent*
   package is what is forbidden, and the check is mechanical:

   ```
   cd go
   AGENTS='camagent lanagent upagent'
   for a in $AGENTS; do
     for p in $(go list ./...); do
       go list -f "{{if ne .Name \"main\"}}{{range .Imports}}{{if eq . \"github.com/jfsmig/cams/go/$a\"}}VIOLATION {{\$.ImportPath}}{{end}}{{end}}{{end}}" $p
     done
   done
   ```
2. **The controller package owns the vocabulary; `agentbus` owns the framing.** The
   vocabulary cannot live in the agent, or every holder of a controller would pull the
   agent in transitively. The framing is shared so that the awkward parts of rule 7 exist
   in exactly one place.
3. **Ownership is a tree, and an owner joins what it started.** `main` owns the upstream
   and LAN agents; the LAN agent owns the camera agents. Every path out of a `Run` has to
   leave nothing of its own behind, which means the teardown belongs in a `defer` rather
   than at the end of the happy path.
4. **An agent depends on no other agent, and on no channel it did not create.** Where it
   needs a neighbour it depends on the interface the neighbour's controller package
   declares -- `camctrl.CameraController`, `lanctrl.LanController` -- and `main` injects
   something that satisfies it. One declaration per contract, beside the vocabulary it
   belongs to, so consumers cannot drift apart. A channel handed in from outside would
   tie two agents' lifetimes together behind the bus's back; there are none, and there is
   no reason to add one.

   **A controller's `New` returns the concrete `*Client`, not the interface.** Returning
   the interface looks tidier and costs more than it looks: the contract is narrower than
   the client on purpose, so returning it puts the remaining methods out of reach of
   every caller including the tests. That is how `Ping`, `State`, `ID` and one `Close`
   went missing once already. Each package carries a
   `var _ XController = (*Client)(nil)` so the two cannot drift.
5. **An agent binds its endpoint when it is built, not when it is run.** A mangos `Dial`
   is synchronous, so a controller created before the agent's `Run` was scheduled would
   be refused. It also means an agent that is built and then abandoned must be `Close`d,
   or its endpoint leaks.
6. **The media does not travel on the control bus.** `agentbus` is REQ/REP and
   synchronous; media is PUSH/PULL through `mediabus`, where the camera is the active
   side and the upstream binds. The queue that bounds the media in flight is the PUSH
   socket's own, and the camera's send is where an overflow is counted.
7. **Requests are at most once.** A `req` socket resends after a minute by default, which
   would replay a command that is not idempotent. Retry is disabled and a receive
   deadline is set instead. The corollary is that a command whose peer disappears while
   answering reports `ErrCanceled`; only a stopping command, where the peer going away
   *is* the result, may treat that as success -- that is what
   `agentbus.Client.DoTolerateGone` is for. A controller fronting an agent that itself
   calls another one wants the longer timeout, so the inner hop expires first and the
   failure is reported where it happened.

## Go engineering rules

These are requirements, not preferences. They follow from what the code is: a
long-running relay, on unattended hardware, fed by devices whose output cannot be
trusted.

### The code has to be unit tested

New or modified logic ships with its tests, in the same change.

* Every package except `go/api/pb` and the two thin `main`s now has tests. Prefer
  adding a test over matching the surrounding absence, and prefer a narrow interface on
  the consumer side over a fixture that needs a server: `upagent` tests its whole gRPC
  logic against `pb.RegistrarClient` and a two-line `controlStream`.
* Anything touching goroutines or channels must also pass `go test -race ./...`.
* Test through the exported API where possible; these packages are small enough that it
  is rarely a real constraint.
* Prefer a table test over a hand-rolled loop, and `t.Fatal` over logging and carrying
  on with a broken fixture.
* When a test's comment and its assertion disagree, the comment usually records the
  intent. Read the implementation before deciding which of the two is the bug.

### Errors are never silently ignored

* No `_ = f()`, no empty `if err != nil {}`, no naked `return` that drops an error. When
  an error truly cannot be acted upon, log it and leave a comment saying why.
* Add context on the way up with `errors.Annotate(err, "describe")` from
  `github.com/juju/errors`, the convention throughout `go/`.
* An error raised inside a goroutine reaches its owner through the group that supervises
  it, not through a log line alone.
* `go vet ./...` must stay clean; it catches the cheapest half of this class.

### No goroutine may leak

Every goroutine has an end that is reachable from the outside.

* A goroutine either takes a `context.Context` and returns when it is done, or belongs
  to an `errgroup.Group` or a `utils.Swarm` that somebody waits on.
* `utils.NewSwarm` and `utils.NewEnsemble` (`go/utils/swarm.go`) are the local idiom for
  a supervised group: they hold the cancel and the `WaitGroup`, and count live members.
* Whoever cancels also waits. A `Cancel()` with no matching `Wait()` lets the process
  tear down while work is still running.
* A library that knows nothing about contexts needs an explicit bridge. See
  `runStreamOnce` in `go/camagent/stream.go`, where a watcher goroutine closes the RTSP
  client in order to release `Client.Wait()`.
* Watch for the deadlock shape where every member of a group waits on a cancellation
  that only that group's own `Wait()` could ever produce.

### No buffer may grow indefinitely

A camera is an adversarial producer by default: it outruns the uplink, or emits
malformed frames forever.

* Every channel crossing a goroutine boundary is created with an explicit capacity, and
  that capacity is a named constant near the type, not a literal at the call site.
* Every queue has a deliberate policy for being full: drop, or apply back-pressure.
  There is no third choice, and "block forever" is not a policy.
* On a media path, dropping beats blocking. Holding a reading goroutine stalls the whole
  RTSP session and trips its timeout detection, which costs far more than one packet.
  Count the drops and expose the counter, the way `mediaSink.push` does in
  `go/camagent/stream.go`.
* Never accumulate per-packet or per-frame data in a slice or a map that nothing prunes.

### Review the local code as a senior Go developer

When reviewing what is already here, apply the judgement of somebody who has operated Go
services, not a linter's.

* Say plainly what is wrong and why it will fail, naming the input or the interleaving
  that breaks it, rather than expressing a general unease.
* Look at concurrency first: races, missed cancellations, unbounded growth, lock
  ordering, a stream that tolerates a single writer being fed from two goroutines.
* Separate what must change from what you would merely prefer, and let the second
  category go.
* Verify a claim against the code before making it. Read the dependency's own source
  when its contract is not obvious: whether a callback fires on every transport is
  answered by the library, not by intuition.
* Report a pre-existing defect found along the way even when it falls outside the task,
  and say that it is pre-existing.

## Build and verify

Everything Go runs from the `go/` subdirectory:

```
cd go
go build ./...
go vet ./...
gofmt -l .            # must print nothing
go test ./...
go test -race ./...   # required for anything touching concurrency
```

`go/go.mod` declares Go **1.26.0** as the language floor, matching the toolchain expected
on a development machine, while the CI image pins a newer release. Keep the floor at or
below the CI version, and do not raise it past the locally installed toolchain without
intending to: with `GOTOOLCHAIN=auto`, Go then downloads a toolchain silently on the next
build.

### Mangos traps, all verified against v3.4.2

The library's documentation disagrees with its code in three places that cost real time.
Each is worked around in `agentbus` or `mediabus`, with a comment; do not undo them.

* **`OptionBestEffort` on PUSH discards about half of everything**, at any load, and
  reports success. It arms the discard branch of a `select` with an already-closed
  channel, so it competes with the enqueue branch even on an empty queue. Measured: of
  2000 frames sent to a puller that was keeping up, 999 arrived.
* **A negative send or receive deadline does not mean "non-blocking"**, contrary to the
  documented behaviour. `xpush` and `xpull` both guard on `> 0`, so a negative value is
  the same as none and blocks forever. There is no non-blocking send.
* **`OptionLinger` is dead code.** The constant is declared and re-exported, with
  upstream's own `// Remove?`, and read by nothing. `Close` never waits, and anything
  still queued is discarded.

Also worth knowing: options must be set before `Dial`/`Listen`, because
`xpush.SetOption` replaces the send queue while `SendMsg` reads it without the lock.

## Running the checks

`ci/go.sh` and `ci/cpp.sh` are what CircleCI and GitHub Actions both run, in the image
from `docker/ci/`. Run those rather than a hand-assembled subset, and add a new check to
the script rather than to one CI configuration.

The C++ half configures from `cpp/`, not from `cpp/hub/`:

```shell
cmake -S cpp -B cpp/build -DCMAKE_BUILD_TYPE=Debug && cmake --build cpp/build
ctest --test-dir cpp/build --output-on-failure
```

It needs FFmpeg 7 or newer (libavformat 61): that is where `avio_alloc_context`'s
`write_packet` callback became unconditionally `const`, which `RtpSource` relies on.
CMake enforces the floor.

There are five `ctest` cases:

* `rtp2hls` replays `cams-capture-879216526.tar` -- a recorded Reolink stream, and the
  reason that file is committed -- through the whole media path and reads the HLS back to
  check it decodes, including across a reconnect.
* `stages` drives `KeyframeGate` and `DurationFiller` directly, with no capture and no
  muxer, which is most of the reason they are separate from the sink.
* `registry` drives `StreamRegistry` with two threads and no gRPC server.
* `storage` drives `StreamStorage::reconcile`, which is pure filesystem.
* `hevc` builds its own H.265 stream, because no recorded HEVC capture exists: libx265
  encodes a few synthetic frames and libavformat's RTP muxer packetises them per RFC
  7798, so the same `run_stream` a camera reaches is exercised. The encoder is a test
  dependency and not a pipeline one; without libx265 the case reports a skip and passes.
  It is **not** a substitute for a real camera -- a device's parameter sets,
  fragmentation and timing are its own -- and the `hvc1` codec tag `FragmentSink` sets
  for HEVC is the one choice in that arm still unverified against a real player.
