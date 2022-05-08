package base

import (
	"context"
)

// Transport is a RTSP transport protocol.
type TransportType int

// transport protocols.
const (
	TransportUDP TransportType = iota
	TransportUDPMulticast
	TransportTCP
)

var transportLabels = map[TransportType]string{
	TransportUDP:          "UDP",
	TransportUDPMulticast: "UDP-multicast",
	TransportTCP:          "TCP",
}

// String implements fmt.Stringer.
func (t TransportType) String() string {
	if l, ok := transportLabels[t]; ok {
		return l
	}
	return "unknown"
}

// Dialer is the factory for RTSP capable devices
type Dialer interface {
	// Dial establishes a communication session with a RTSP capable device, given its
	// address, within a deadline
	Dial(ctx context.Context, address string) Conn
}
