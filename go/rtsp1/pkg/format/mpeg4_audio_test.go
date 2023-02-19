package format

import (
	mpeg4audio2 "github.com/jfsmig/streaming/rtsp1/pkg/codecs/mpeg4audio"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMPEG4AudioAttributes(t *testing.T) {
	format := &MPEG4Audio{
		PayloadTyp: 96,
		Config: &mpeg4audio2.Config{
			Type:         mpeg4audio2.ObjectTypeAACLC,
			SampleRate:   48000,
			ChannelCount: 2,
		},
		SizeLength:       13,
		IndexLength:      3,
		IndexDeltaLength: 3,
	}
	require.Equal(t, "MPEG4-audio", format.String())
	require.Equal(t, 48000, format.ClockRate())
	require.Equal(t, uint8(96), format.PayloadType())
}

func TestMPEG4AudioMediaDescription(t *testing.T) {
	format := &MPEG4Audio{
		PayloadTyp: 96,
		Config: &mpeg4audio2.Config{
			Type:         mpeg4audio2.ObjectTypeAACLC,
			SampleRate:   48000,
			ChannelCount: 2,
		},
		SizeLength:       13,
		IndexLength:      3,
		IndexDeltaLength: 3,
	}

	rtpmap, fmtp := format.Marshal()
	require.Equal(t, "mpeg4-generic/48000/2", rtpmap)
	require.Equal(t, "profile-level-id=1; mode=AAC-hbr; sizelength=13;"+
		" indexlength=3; indexdeltalength=3; config=1190", fmtp)
}
