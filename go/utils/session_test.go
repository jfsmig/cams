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

package utils

import (
	"context"
	"testing"

	"google.golang.org/grpc/metadata"
)

func TestSessionID_IsSetAndStable(t *testing.T) {
	if SessionID == "" {
		t.Fatal("the session id is empty, so every log line joins on nothing")
	}
	// Two draws must differ, or a fleet of agents shares one id and the
	// column stops distinguishing anything.
	if a, b := newSessionID(), newSessionID(); a == b {
		t.Fatalf("two session ids are identical: %q", a)
	}
}

func TestWithSession_TagsAnOutgoingCall(t *testing.T) {
	md, ok := metadata.FromOutgoingContext(WithSession(context.Background()))
	if !ok {
		t.Fatal("no outgoing metadata at all")
	}
	got := md.Get(KeySession)
	if len(got) != 1 || got[0] != SessionID {
		t.Fatalf("session metadata is %v, want [%s]", got, SessionID)
	}
}

// TestWithSession_KeepsWhatWasThere is the ordering trap: the registrar builds
// its metadata block with NewOutgoingContext, which replaces rather than adds,
// so the tag has to be applied to the result.
func TestWithSession_KeepsWhatWasThere(t *testing.T) {
	ctx := metadata.NewOutgoingContext(context.Background(),
		metadata.New(map[string]string{KeyUser: "someone"}))

	md, ok := metadata.FromOutgoingContext(WithSession(ctx))
	if !ok {
		t.Fatal("no outgoing metadata at all")
	}
	if u := md.Get(KeyUser); len(u) != 1 || u[0] != "someone" {
		t.Fatalf("the user was lost: %v", u)
	}
	if s := md.Get(KeySession); len(s) != 1 || s[0] != SessionID {
		t.Fatalf("session metadata is %v, want [%s]", s, SessionID)
	}
}
