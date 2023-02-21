package mpeg4audio

import (
	"fmt"
	bits2 "github.com/jfsmig/cams/go/rtsp1/pkg/bits"
)

// Config is a MPEG-4 Audio configuration.
type Config struct {
	Type         ObjectType
	SampleRate   int
	ChannelCount int

	// SBR / PS specific
	ExtensionType       ObjectType
	ExtensionSampleRate int

	// AAC-LC specific
	FrameLengthFlag    bool
	DependsOnCoreCoder bool
	CoreCoderDelay     uint16
}

// Unmarshal decodes a Config.
func (c *Config) Unmarshal(buf []byte) error {
	pos := 0

	tmp, err := bits2.ReadBits(buf, &pos, 5)
	if err != nil {
		return err
	}
	c.Type = ObjectType(tmp)

	switch c.Type {
	case ObjectTypeAACLC:
	case ObjectTypeSBR:
	case ObjectTypePS:
	default:
		return fmt.Errorf("unsupported object type: %d", c.Type)
	}

	sampleRateIndex, err := bits2.ReadBits(buf, &pos, 4)
	if err != nil {
		return err
	}

	switch {
	case sampleRateIndex <= 12:
		c.SampleRate = sampleRates[sampleRateIndex]

	case sampleRateIndex == 0x0F:
		tmp, err := bits2.ReadBits(buf, &pos, 24)
		if err != nil {
			return err
		}
		c.SampleRate = int(tmp)

	default:
		return fmt.Errorf("invalid sample rate index (%d)", sampleRateIndex)
	}

	channelConfig, err := bits2.ReadBits(buf, &pos, 4)
	if err != nil {
		return err
	}

	switch {
	case channelConfig == 0:
		return fmt.Errorf("not yet supported")

	case channelConfig >= 1 && channelConfig <= 6:
		c.ChannelCount = int(channelConfig)

	case channelConfig == 7:
		c.ChannelCount = 8

	default:
		return fmt.Errorf("invalid channel configuration (%d)", channelConfig)
	}

	if c.Type == ObjectTypeSBR || c.Type == ObjectTypePS {
		c.ExtensionType = c.Type
		extensionSamplingFrequencyIndex, err := bits2.ReadBits(buf, &pos, 4)
		if err != nil {
			return err
		}

		switch {
		case extensionSamplingFrequencyIndex <= 12:
			c.ExtensionSampleRate = sampleRates[extensionSamplingFrequencyIndex]

		case extensionSamplingFrequencyIndex == 0x0F:
			tmp, err := bits2.ReadBits(buf, &pos, 24)
			if err != nil {
				return err
			}
			c.ExtensionSampleRate = int(tmp)

		default:
			return fmt.Errorf("invalid extension sample rate index (%d)", extensionSamplingFrequencyIndex)
		}

		tmp, err = bits2.ReadBits(buf, &pos, 5)
		if err != nil {
			return err
		}
		c.Type = ObjectType(tmp)

		if c.Type != ObjectTypeAACLC {
			return fmt.Errorf("unsupported object type: %d", c.Type)
		}
	}

	c.FrameLengthFlag, err = bits2.ReadFlag(buf, &pos)
	if err != nil {
		return err
	}

	c.DependsOnCoreCoder, err = bits2.ReadFlag(buf, &pos)
	if err != nil {
		return err
	}

	if c.DependsOnCoreCoder {
		tmp, err := bits2.ReadBits(buf, &pos, 14)
		if err != nil {
			return err
		}
		c.CoreCoderDelay = uint16(tmp)
	}

	extensionFlag, err := bits2.ReadFlag(buf, &pos)
	if err != nil {
		return err
	}

	if extensionFlag {
		return fmt.Errorf("unsupported")
	}

	return nil
}

func (c Config) marshalSize() int {
	n := 5 + 4 + 3

	_, ok := reverseSampleRates[c.SampleRate]
	if !ok {
		n += 28
	} else {
		n += 4
	}

	if c.ExtensionType == ObjectTypeSBR || c.ExtensionType == ObjectTypePS {
		_, ok := reverseSampleRates[c.ExtensionSampleRate]
		if !ok {
			n += 28
		} else {
			n += 4
		}
		n += 5
	} else if c.DependsOnCoreCoder {
		n += 14
	}

	ret := n / 8
	if (n % 8) != 0 {
		ret++
	}

	return ret
}

// Marshal encodes a Config.
func (c Config) Marshal() ([]byte, error) {
	buf := make([]byte, c.marshalSize())
	pos := 0

	if c.ExtensionType == ObjectTypeSBR || c.ExtensionType == ObjectTypePS {
		bits2.WriteBits(buf, &pos, uint64(c.ExtensionType), 5)
	} else {
		bits2.WriteBits(buf, &pos, uint64(c.Type), 5)
	}

	sampleRateIndex, ok := reverseSampleRates[c.SampleRate]
	if !ok {
		bits2.WriteBits(buf, &pos, uint64(15), 4)
		bits2.WriteBits(buf, &pos, uint64(c.SampleRate), 24)
	} else {
		bits2.WriteBits(buf, &pos, uint64(sampleRateIndex), 4)
	}

	var channelConfig int
	switch {
	case c.ChannelCount >= 1 && c.ChannelCount <= 6:
		channelConfig = c.ChannelCount

	case c.ChannelCount == 8:
		channelConfig = 7

	default:
		return nil, fmt.Errorf("invalid channel count (%d)", c.ChannelCount)
	}
	bits2.WriteBits(buf, &pos, uint64(channelConfig), 4)

	if c.ExtensionType == ObjectTypeSBR || c.ExtensionType == ObjectTypePS {
		sampleRateIndex, ok := reverseSampleRates[c.ExtensionSampleRate]
		if !ok {
			bits2.WriteBits(buf, &pos, uint64(0x0F), 4)
			bits2.WriteBits(buf, &pos, uint64(c.ExtensionSampleRate), 24)
		} else {
			bits2.WriteBits(buf, &pos, uint64(sampleRateIndex), 4)
		}
		bits2.WriteBits(buf, &pos, uint64(c.Type), 5)
	} else {
		if c.FrameLengthFlag {
			bits2.WriteBits(buf, &pos, 1, 1)
		} else {
			bits2.WriteBits(buf, &pos, 0, 1)
		}

		if c.DependsOnCoreCoder {
			bits2.WriteBits(buf, &pos, 1, 1)
		} else {
			bits2.WriteBits(buf, &pos, 0, 1)
		}

		if c.DependsOnCoreCoder {
			bits2.WriteBits(buf, &pos, uint64(c.CoreCoderDelay), 14)
		}
	}

	return buf, nil
}