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
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/jfsmig/cams/go/api/pb"
	"github.com/jfsmig/cams/go/utils"
	"github.com/juju/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// freeAddr reserves an ephemeral port and hands back its address, so that a
// test can name a port it knows is free without racing another test on a fixed
// one.
func freeAddr(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve a port: %v", err)
	}
	addr := l.Addr().String()
	if err := l.Close(); err != nil {
		t.Fatalf("release the port: %v", err)
	}
	return addr
}

// TestRunHub_Starts is the regression test for the defect that made the binary
// unusable: two listeners were opened on the same address, so the second failed
// with EADDRINUSE and runHub returned before serving anything.
func TestRunHub_Starts(t *testing.T) {
	addr := freeAddr(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- runHub(ctx, utils.ServerConfig{ListenAddr: addr})
	}()

	// A hub that failed to bind returns immediately with an error.
	select {
	case err := <-done:
		t.Fatalf("runHub returned before serving: %v", err)
	case <-time.After(250 * time.Millisecond):
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runHub: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("runHub did not return after the context was cancelled")
	}
}

// TestRunHub_ServesEveryService checks that all four services answer on the one
// port. They used to be split across two servers bound to the same address,
// which is why only the collapse to a single listener makes this pass.
func TestRunHub_ServesEveryService(t *testing.T) {
	addr := freeAddr(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- runHub(ctx, utils.ServerConfig{ListenAddr: addr})
	}()
	defer func() {
		cancel()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Error("runHub did not return")
		}
	}()

	dialCtx, dialCancel := context.WithTimeout(ctx, 5*time.Second)
	defer dialCancel()

	cnx, err := utils.DialInsecure(dialCtx, addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() {
		if cerr := cnx.Close(); cerr != nil {
			t.Errorf("close: %v", cerr)
		}
	}()

	// The agents send the account as metadata, and the hub now requires it.
	authCtx := metadata.AppendToOutgoingContext(dialCtx, utils.KeyUser, "someone")

	// Registrar: a well-formed registration has to be accepted.
	reg := pb.NewRegistrarClient(cnx)
	if _, err := reg.Register(authCtx, &pb.RegisterRequest{
		Id: &pb.StreamId{User: "someone", Stream: "cam-1"},
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Without it, refused rather than believed.
	if _, err := reg.Register(dialCtx, &pb.RegisterRequest{
		Id: &pb.StreamId{User: "someone", Stream: "cam-x"},
	}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("an unauthenticated registration gave %v, want InvalidArgument", err)
	}

	// And a body that names a different account than the caller is refused: it
	// used to be taken at face value, so any peer could register under any name.
	if _, err := reg.Register(authCtx, &pb.RegisterRequest{
		Id: &pb.StreamId{User: "someone-else", Stream: "cam-y"},
	}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("a registration for another account gave %v, want PermissionDenied", err)
	}

	// Viewer: reachable, and honest about an agent that is not connected.
	view := pb.NewViewerClient(cnx)
	_, err = view.Play(dialCtx, &pb.PlayRequest{
		Id: &pb.StreamId{User: "nobody", Stream: "cam-1"},
	})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("Play for an unconnected agent gave %v, want NotFound", err)
	}

	// Uploader: reachable, and refuses instead of taking the hub down.
	up := pb.NewUploaderClient(cnx)
	stream, err := up.MediaUpload(dialCtx)
	if err != nil {
		t.Fatalf("MediaUpload: %v", err)
	}
	if err := stream.Send(&pb.DownstreamMediaFrame{
		Type:    pb.DownstreamMediaFrameType_DOWNSTREAM_MEDIA_FRAME_TYPE_SDP,
		Payload: []byte("v=0"),
	}); err != nil && status.Code(err) != codes.Unimplemented {
		t.Fatalf("Send: %v", err)
	}
	if _, err := stream.CloseAndRecv(); status.Code(err) != codes.Unimplemented {
		t.Fatalf("MediaUpload gave %v, want Unimplemented", err)
	}

	// And the hub is still serving afterwards, which is the point: the media
	// endpoint used to panic and take the control plane with it.
	if _, err := reg.Register(authCtx, &pb.RegisterRequest{
		Id: &pb.StreamId{User: "someone", Stream: "cam-2"},
	}); err != nil {
		t.Fatalf("the hub stopped serving after a media upload: %v", err)
	}
}

// TestStreamIdOf covers the nil dereference that a two-field request was enough
// to trigger, on both the registrar and the viewer.
func TestStreamIdOf(t *testing.T) {
	if _, err := streamIdOf(&pb.StreamId{User: "u", Stream: "s"}); err != nil {
		t.Fatalf("a complete identity was refused: %v", err)
	}

	for _, tc := range []struct {
		name string
		id   *pb.StreamId
	}{
		{"nil", nil},
		{"no user", &pb.StreamId{Stream: "s"}},
		{"no stream", &pb.StreamId{User: "u"}},
		{"empty", &pb.StreamId{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := streamIdOf(tc.id); err == nil {
				t.Fatal("accepted")
			} else if status.Code(err) != codes.InvalidArgument {
				t.Fatalf("gave %v, want InvalidArgument", err)
			}
		})
	}
}

// TestRegister_RejectsAnIncompleteRequest is the same defect reached through
// the RPC, since that is where a malformed request actually arrives.
func TestRegister_RejectsAnIncompleteRequest(t *testing.T) {
	hub := &grpcHub{registrar: NewRegistrarInMem()}

	ctx := metadata.NewIncomingContext(context.Background(),
		metadata.New(map[string]string{utils.KeyUser: "someone"}))
	if _, err := hub.Register(ctx, &pb.RegisterRequest{}); err == nil {
		t.Fatal("a registration with no identity was accepted")
	} else if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("gave %v, want InvalidArgument", err)
	}
}

// ---------- the agent registry ----------

func TestAgents_AddGetRemove(t *testing.T) {
	hub := &grpcHub{}

	if got := hub.getAgent("someone"); got != nil {
		t.Fatalf("getAgent on an empty hub = %v, want nil", got)
	}

	first := NewAgentTwin("someone")
	if !hub.addAgent(first) {
		t.Fatal("the first agent was refused")
	}
	if got := hub.getAgent("someone"); got != first {
		t.Fatalf("getAgent = %v, want the agent just added", got)
	}

	// A second connection for the same identity has to be refused, and must
	// not replace the one in place.
	if hub.addAgent(NewAgentTwin("someone")) {
		t.Fatal("a duplicate agent was accepted")
	}
	if got := hub.getAgent("someone"); got != first {
		t.Fatal("a refused duplicate replaced the agent in place")
	}

	hub.removeAgent("someone")
	if got := hub.getAgent("someone"); got != nil {
		t.Fatalf("getAgent after removal = %v, want nil", got)
	}
}

// TestAgents_Concurrent is why the map has a lock: one goroutine per connected
// agent adds and removes while every viewer RPC reads.
func TestAgents_Concurrent(t *testing.T) {
	hub := &grpcHub{}

	const agents = 8
	const rounds = 50

	stop := make(chan struct{})
	readers := make(chan struct{}, 2)
	for r := 0; r < 2; r++ {
		go func() {
			defer func() { readers <- struct{}{} }()
			for {
				select {
				case <-stop:
					return
				default:
					hub.getAgent(AgentID("agent-0"))
				}
			}
		}()
	}

	done := make(chan struct{}, agents)
	for a := 0; a < agents; a++ {
		go func(a int) {
			defer func() { done <- struct{}{} }()
			id := AgentID("agent-" + string(rune('0'+a)))
			for i := 0; i < rounds; i++ {
				if hub.addAgent(NewAgentTwin(id)) {
					hub.removeAgent(id)
				}
			}
		}(a)
	}
	for a := 0; a < agents; a++ {
		<-done
	}
	close(stop)
	<-readers
	<-readers
}

// ---------- the agent twin ----------

// TestAgentTwin_RefusesWhenGone covers the send on a closed channel: the twin's
// request channel used to be closed when its stream ended, so a viewer RPC that
// was mid-send at that moment panicked the hub.
func TestAgentTwin_RefusesWhenGone(t *testing.T) {
	agent := NewAgentTwin("someone")
	agent.close()

	err := agent.Play(context.Background(), "cam-1")
	if err == nil {
		t.Fatal("a command for a departed agent was accepted")
	}
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("gave %v, want Unavailable", err)
	}
	if err := agent.Stop(context.Background(), "cam-1"); status.Code(err) != codes.Unavailable {
		t.Fatalf("Stop gave %v, want Unavailable", err)
	}
}

// TestAgentTwin_GivesUpWithTheCaller checks that a slow agent cannot hold a
// viewer RPC: the queue has room for one, so the second send has to wait, and
// it waits no longer than its caller does.
func TestAgentTwin_GivesUpWithTheCaller(t *testing.T) {
	agent := NewAgentTwin("someone")

	// Fill the single slot.
	if err := agent.Play(context.Background(), "cam-1"); err != nil {
		t.Fatalf("the first command was refused: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := agent.Play(ctx, "cam-2")
	if err == nil {
		t.Fatal("a second command was accepted with a full queue and no reader")
	}
	if status.Code(err) != codes.DeadlineExceeded {
		t.Fatalf("gave %v, want DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("waited %v, far longer than the caller's deadline", elapsed)
	}
}

// TestAgentTwin_DeliversToAReader is the happy path, so the refusals above are
// not passing for the wrong reason.
func TestAgentTwin_DeliversToAReader(t *testing.T) {
	agent := NewAgentTwin("someone")

	if err := agent.Play(context.Background(), "cam-1"); err != nil {
		t.Fatalf("Play: %v", err)
	}

	select {
	case cmd := <-agent.requests:
		if cmd.cmdType != CtrlCommandType_Play || cmd.streamID != "cam-1" {
			t.Fatalf("received %v", cmd)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the command never arrived")
	}
}

// ---------- the error boundary ----------

// TestStatusOf pins the mapping from the registry's juju error types onto gRPC
// codes. It matters that these predicates really fire: if juju's sentinels did
// not match, every domain failure would silently become codes.Internal.
func TestStatusOf(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want codes.Code
	}{
		{"already exists", errors.AlreadyExistsf("stream %q", "cam-1"), codes.AlreadyExists},
		{"not found", errors.NotFoundf("stream %q", "cam-1"), codes.NotFound},
		{"not valid", errors.NotValidf("stream %q", "cam-1"), codes.InvalidArgument},
		{"not supported", errors.NotSupportedf("verb %q", "PIROUETTE"), codes.Unimplemented},
		{"anything else", errors.New("the disk is on fire"), codes.Internal},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := status.Code(statusOf(tc.err)); got != tc.want {
				t.Fatalf("statusOf(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}

	if statusOf(nil) != nil {
		t.Fatal("statusOf(nil) invented an error")
	}

	// An error that already carries a status keeps it.
	given := status.Error(codes.PermissionDenied, "no")
	if got := status.Code(statusOf(given)); got != codes.PermissionDenied {
		t.Fatalf("statusOf re-coded an existing status as %v", got)
	}
}

// TestUserOf covers the identity lookup that both Control and Register use, and
// with it the index-without-a-length-check that any peer could panic.
func TestUserOf(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(),
		metadata.New(map[string]string{utils.KeyUser: "someone"}))
	if got, err := userOf(ctx); err != nil || got != "someone" {
		t.Fatalf("userOf = (%q, %v)", got, err)
	}

	for _, tc := range []struct {
		name string
		ctx  context.Context
	}{
		{"no metadata", context.Background()},
		{"no key", metadata.NewIncomingContext(context.Background(), metadata.New(nil))},
		{"empty value", metadata.NewIncomingContext(context.Background(),
			metadata.New(map[string]string{utils.KeyUser: ""}))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := userOf(tc.ctx); err == nil {
				t.Fatal("accepted")
			} else if status.Code(err) != codes.InvalidArgument {
				t.Fatalf("gave %v, want InvalidArgument", err)
			}
		})
	}
}

// ---------- the registry's bounds ----------

func TestRegistrar_RefusesAnotherUsersStream(t *testing.T) {
	r := NewRegistrarInMem()

	if err := r.Register(StreamRegistration{StreamID: "cam-1", User: "alice"}); err != nil {
		t.Fatalf("first registration: %v", err)
	}
	// The same owner re-announcing is the normal case.
	if err := r.Register(StreamRegistration{StreamID: "cam-1", User: "alice"}); err != nil {
		t.Fatalf("re-registration: %v", err)
	}

	err := r.Register(StreamRegistration{StreamID: "cam-1", User: "bob"})
	if err == nil {
		t.Fatal("a stream was taken over by another user")
	}
	// And the code reaches the client as something it can act on.
	if got := status.Code(statusOf(err)); got != codes.AlreadyExists {
		t.Fatalf("gave %v, want AlreadyExists", got)
	}
}

// TestRegistrar_IsBounded is the regression test for unbounded growth: Register
// is reachable by any peer, and every call used to add an entry that nothing
// ever removed.
func TestRegistrar_IsBounded(t *testing.T) {
	r := NewRegistrarInMem().(*registrarInMem)
	// A small cap: the production one is 10000 and the set re-sorts on every
	// insert, so filling it would dominate the whole suite.
	r.maxStreams = 16

	for i := 0; i < r.maxStreams; i++ {
		if err := r.Register(StreamRegistration{StreamID: fmt.Sprintf("cam-%d", i), User: "alice"}); err != nil {
			t.Fatalf("registration %d: %v", i, err)
		}
	}
	if err := r.Register(StreamRegistration{StreamID: "one-too-many", User: "alice"}); err == nil {
		t.Fatal("the registry accepted an entry past its cap")
	}
	if len(r.streams) > r.maxStreams {
		t.Fatalf("the registry holds %d entries, past the cap of %d", len(r.streams), r.maxStreams)
	}
}

// TestRegistrar_ExpiresWhatIsStale checks that latUpdate is finally read: it
// was written on every registration and consulted nowhere, so nothing aged out.
func TestRegistrar_ExpiresWhatIsStale(t *testing.T) {
	r := NewRegistrarInMem().(*registrarInMem)

	if err := r.Register(StreamRegistration{StreamID: "old", User: "alice"}); err != nil {
		t.Fatalf("registration: %v", err)
	}
	if err := r.Register(StreamRegistration{StreamID: "fresh", User: "alice"}); err != nil {
		t.Fatalf("registration: %v", err)
	}

	// Age one of them past the window.
	r.lock.Lock()
	sr, ok := r.streams.Get("old")
	if !ok {
		r.lock.Unlock()
		t.Fatal("the entry was not registered")
	}
	sr.latUpdate = time.Now().Add(-2 * registrationTTL)
	r.lock.Unlock()

	// Any first sighting sweeps.
	if err := r.Register(StreamRegistration{StreamID: "trigger", User: "alice"}); err != nil {
		t.Fatalf("registration: %v", err)
	}

	r.lock.Lock()
	defer r.lock.Unlock()
	if _, ok := r.streams.Get("old"); ok {
		t.Fatal("a stale entry survived the sweep")
	}
	if _, ok := r.streams.Get("fresh"); !ok {
		t.Fatal("the sweep took a fresh entry with it")
	}
	if _, ok := r.streams.Get("trigger"); !ok {
		t.Fatal("the entry that triggered the sweep was not kept")
	}
}
