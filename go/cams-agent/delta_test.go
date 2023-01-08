package main

import (
	"github.com/jfsmig/cams/go/cams-agent/lan"
	"testing"
)

func TestDelta(t *testing.T) {
	if 4 != lan.delta[uint32](2, lan.maxValue[uint32]()-2) {
		t.Fatal()
	}
	if 4 != lan.delta[uint32](6, 2) {
		t.Fatal()
	}
}
