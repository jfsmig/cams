package camera

import (
	"context"
	"github.com/aler9/gortsplib"
)

type UpstreamMedia interface {
	Close()
	OnSDP(sdp []byte) error
	OnRTP(pkt *gortsplib.ClientOnPacketRTPCtx) error
	OnRTCP(pkt *gortsplib.ClientOnPacketRTCPCtx) error
}

type UploadOpenFunc func(ctx context.Context) (UpstreamMedia, error)
