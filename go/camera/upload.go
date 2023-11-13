package camera

import (
	"context"
)

type UpstreamMedia interface {
	Close()
	OnSDP(sdp string) error
	OnRTP(pkt []byte) error
	OnRTCP(pkt []byte) error
}

type UploadOpenFunc func(ctx context.Context) (UpstreamMedia, error)
