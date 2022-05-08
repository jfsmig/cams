package format

// G722 is a G722 format.
type G722 struct{}

// String implements Format.
func (t *G722) String() string {
	return "G722"
}

// ClockRate implements Format.
func (t *G722) ClockRate() int {
	return 8000
}

// PayloadType implements Format.
func (t *G722) PayloadType() uint8 {
	return 9
}

func (t *G722) unmarshal(payloadType uint8, clock string, codec string, rtpmap string, fmtp string) error {
	return nil
}

// Marshal implements Format.
func (t *G722) Marshal() (string, string) {
	return "G722/8000", ""
}
