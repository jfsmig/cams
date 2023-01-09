package main

import (
	"testing"
)

func TestDelta(t *testing.T) {
	if 4 != delta[uint32](2, maxValue[uint32]()-2) {
		t.Fatal()
	}
	if 4 != delta[uint32](6, 2) {
		t.Fatal()
	}
}
