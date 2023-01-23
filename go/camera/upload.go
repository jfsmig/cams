package camera

import (
	"context"
	"github.com/aler9/gortsplib/v2/pkg/format"
	"github.com/aler9/gortsplib/v2/pkg/media"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
)

type UpstreamMedia interface {
	Close()
	OnSDP(sdp []byte) error
	OnRTP(m *media.Media, f format.Format, pkt *rtp.Packet) error
	OnRTCP(m *media.Media, pkt *rtcp.Packet) error
}

type UploadOpenFunc func(ctx context.Context) (UpstreamMedia, error)
