package format

import (
	"fmt"
	"strconv"
	"strings"
)

// Opus is a Opus format.
type Opus struct {
	PayloadTyp   uint8
	SampleRate   int
	ChannelCount int
}

// String implements Format.
func (t *Opus) String() string {
	return "Opus"
}

// ClockRate implements Format.
func (t *Opus) ClockRate() int {
	return t.SampleRate
}

// PayloadType implements Format.
func (t *Opus) PayloadType() uint8 {
	return t.PayloadTyp
}

func (t *Opus) unmarshal(payloadType uint8, clock string, codec string, rtpmap string, fmtp string) error {
	t.PayloadTyp = payloadType

	tmp := strings.SplitN(clock, "/", 2)
	if len(tmp) != 2 {
		return fmt.Errorf("invalid clock (%v)", clock)
	}

	sampleRate, err := strconv.ParseInt(tmp[0], 10, 64)
	if err != nil {
		return err
	}
	t.SampleRate = int(sampleRate)

	channelCount, err := strconv.ParseInt(tmp[1], 10, 64)
	if err != nil {
		return err
	}
	t.ChannelCount = int(channelCount)

	return nil
}

// Marshal implements Format.
func (t *Opus) Marshal() (string, string) {
	fmtp := "sprop-stereo=" + func() string {
		if t.ChannelCount == 2 {
			return "1"
		}
		return "0"
	}()

	return "opus/" + strconv.FormatInt(int64(t.SampleRate), 10) +
		"/" + strconv.FormatInt(int64(t.ChannelCount), 10), fmtp
}
