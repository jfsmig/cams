package format

// G711 is a G711 format, encoded with mu-law or A-law.
type G711 struct {
	// whether to use mu-law. Otherwise, A-law is used.
	MULaw bool
}

// String implements Format.
func (t *G711) String() string {
	return "G711"
}

// ClockRate implements Format.
func (t *G711) ClockRate() int {
	return 8000
}

// PayloadType implements Format.
func (t *G711) PayloadType() uint8 {
	if t.MULaw {
		return 0
	}
	return 8
}

func (t *G711) unmarshal(payloadType uint8, clock string, codec string, rtpmap string, fmtp string) error {
	t.MULaw = (payloadType == 0)
	return nil
}

// Marshal implements Format.
func (t *G711) Marshal() (string, string) {
	if t.MULaw {
		return "PCMU/8000", ""
	}
	return "PCMA/8000", ""
}
