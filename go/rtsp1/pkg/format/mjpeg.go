package format

// MJPEG is a Motion-JPEG format.
type MJPEG struct{}

// String implements Format.
func (t *MJPEG) String() string {
	return "M-JPEG"
}

// ClockRate implements Format.
func (t *MJPEG) ClockRate() int {
	return 90000
}

// PayloadType implements Format.
func (t *MJPEG) PayloadType() uint8 {
	return 26
}

func (t *MJPEG) unmarshal(payloadType uint8, clock string, codec string, rtpmap string, fmtp string) error {
	return nil
}

// Marshal implements Format.
func (t *MJPEG) Marshal() (string, string) {
	return "JPEG/90000", ""
}
