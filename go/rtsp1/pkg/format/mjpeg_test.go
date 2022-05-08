package format

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMJPEGAttributes(t *testing.T) {
	format := &MJPEG{}
	require.Equal(t, "M-JPEG", format.String())
	require.Equal(t, 90000, format.ClockRate())
	require.Equal(t, uint8(26), format.PayloadType())
}

func TestMJPEGMediaDescription(t *testing.T) {
	format := &MJPEG{}

	rtpmap, fmtp := format.Marshal()
	require.Equal(t, "JPEG/90000", rtpmap)
	require.Equal(t, "", fmtp)
}
