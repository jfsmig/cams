//go:build go1.18
// +build go1.18

package h264

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var casesAnnexB = []struct {
	name   string
	encin  []byte
	encout []byte
	dec    [][]byte
}{
	{
		"2 zeros",
		[]byte{
			0x00, 0x00, 0x01, 0xaa, 0xbb, 0x00, 0x00, 0x01,
			0xcc, 0xdd, 0x00, 0x00, 0x01, 0xee, 0xff,
		},
		[]byte{
			0x00, 0x00, 0x00, 0x01, 0xaa, 0xbb,
			0x00, 0x00, 0x00, 0x01, 0xcc, 0xdd,
			0x00, 0x00, 0x00, 0x01, 0xee, 0xff,
		},
		[][]byte{
			{0xaa, 0xbb},
			{0xcc, 0xdd},
			{0xee, 0xff},
		},
	},
	{
		"3 zeros",
		[]byte{
			0x00, 0x00, 0x00, 0x01, 0xaa, 0xbb,
			0x00, 0x00, 0x00, 0x01, 0xcc, 0xdd,
			0x00, 0x00, 0x00, 0x01, 0xee, 0xff,
		},
		[]byte{
			0x00, 0x00, 0x00, 0x01, 0xaa, 0xbb,
			0x00, 0x00, 0x00, 0x01, 0xcc, 0xdd,
			0x00, 0x00, 0x00, 0x01, 0xee, 0xff,
		},
		[][]byte{
			{0xaa, 0xbb},
			{0xcc, 0xdd},
			{0xee, 0xff},
		},
	},
	{
		// used by Apple inside HLS test streams
		"2 or 3 zeros",
		[]byte{
			0, 0, 0, 1, 9, 240,
			0, 0, 0, 1, 39, 66, 224, 21, 169, 24, 60, 23, 252, 184, 3, 80, 96, 16, 107, 108, 43, 94, 247, 192, 64,
			0, 0, 0, 1, 40, 222, 9, 200,
			0, 0, 1, 6, 0, 7, 131, 236, 119, 0, 0, 0, 0, 1, 3, 0, 64, 128,
			0, 0, 1, 6, 5, 17, 3, 135, 244, 78, 205, 10, 75, 220, 161, 148, 58, 195, 212, 155, 23, 31, 0, 128,
		},
		[]byte{
			0, 0, 0, 1, 9, 240,
			0, 0, 0, 1, 39, 66, 224, 21, 169, 24, 60, 23, 252, 184, 3, 80, 96, 16, 107, 108, 43, 94, 247, 192, 64,
			0, 0, 0, 1, 40, 222, 9, 200,
			0, 0, 0, 1, 6, 0, 7, 131, 236, 119, 0, 0, 0, 0, 1, 3, 0, 64, 128,
			0, 0, 0, 1, 6, 5, 17, 3, 135, 244, 78, 205, 10, 75, 220, 161, 148, 58, 195, 212, 155, 23, 31, 0, 128,
		},
		[][]byte{
			{9, 240},
			{39, 66, 224, 21, 169, 24, 60, 23, 252, 184, 3, 80, 96, 16, 107, 108, 43, 94, 247, 192, 64},
			{40, 222, 9, 200},
			{6, 0, 7, 131, 236, 119, 0, 0, 0, 0, 1, 3, 0, 64, 128},
			{6, 5, 17, 3, 135, 244, 78, 205, 10, 75, 220, 161, 148, 58, 195, 212, 155, 23, 31, 0, 128},
		},
	},
}

func TestAnnexBUnmarshal(t *testing.T) {
	for _, ca := range casesAnnexB {
		t.Run(ca.name, func(t *testing.T) {
			dec, err := AnnexBUnmarshal(ca.encin)
			require.NoError(t, err)
			require.Equal(t, ca.dec, dec)
		})
	}
}

func TestAnnexBMarshal(t *testing.T) {
	for _, ca := range casesAnnexB {
		t.Run(ca.name, func(t *testing.T) {
			enc, err := AnnexBMarshal(ca.dec)
			require.NoError(t, err)
			require.Equal(t, ca.encout, enc)
		})
	}
}

func BenchmarkAnnexBUnmarshal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		AnnexBUnmarshal([]byte{
			0x00, 0x00, 0x00, 0x01,
			0x01, 0x02, 0x03, 0x04,
			0x00, 0x00, 0x00, 0x01,
			0x01, 0x02, 0x03, 0x04,
			0x00, 0x00, 0x00, 0x01,
			0x01, 0x02, 0x03, 0x04,
			0x00, 0x00, 0x00, 0x01,
			0x01, 0x02, 0x03, 0x04,
			0x00, 0x00, 0x00, 0x01,
			0x01, 0x02, 0x03, 0x04,
			0x00, 0x00, 0x00, 0x01,
			0x01, 0x02, 0x03, 0x04,
			0x00, 0x00, 0x00, 0x01,
			0x01, 0x02, 0x03, 0x04,
			0x00, 0x00, 0x00, 0x01,
			0x01, 0x02, 0x03, 0x04,
		})
	}
}

func FuzzAnnexBUnmarshal(f *testing.F) {
	f.Fuzz(func(t *testing.T, b []byte) {
		AnnexBUnmarshal(b)
	})
}
