package format

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLPCMAttributes(t *testing.T) {
	format := &LPCM{
		PayloadTyp:   96,
		BitDepth:     24,
		SampleRate:   44100,
		ChannelCount: 2,
	}
	require.Equal(t, "LPCM", format.String())
	require.Equal(t, 44100, format.ClockRate())
	require.Equal(t, uint8(96), format.PayloadType())
}

func TestLPCMMediaDescription(t *testing.T) {
	for _, ca := range []int{8, 16, 24} {
		t.Run(strconv.FormatInt(int64(ca), 10), func(t *testing.T) {
			format := &LPCM{
				PayloadTyp:   96,
				BitDepth:     ca,
				SampleRate:   96000,
				ChannelCount: 2,
			}

			rtpmap, fmtp := format.Marshal()
			require.Equal(t, fmt.Sprintf("L%d/96000/2", ca), rtpmap)
			require.Equal(t, "", fmtp)
		})
	}
}
