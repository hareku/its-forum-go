package rc013

// Message is a standalone RC-013 version-1 basic message. Header.DataLength and
// Header.OptionFlags are observations after decoding; MarshalBinary derives them
// from Common and Free. Free nil represents absence, not a zero-entry header.
type Message struct {
	Header       Header
	Common       CommonData
	Free         *FreeArea
	freeBase     int
	freeAnchored bool
}

func supportedHeader(h Header) error {
	if h.ServiceID != 1 || h.MessageID != 1 || h.Version != 1 {
		return fieldError("header.identifiers", 0, ErrUnsupported)
	}
	return nil
}

// Decode reads exactly one 36..100-byte message and takes ownership of copies of
// all retained bytes. Unknown scalar values remain readable; Validate is separate.
func Decode(data []byte) (Message, error) {
	if len(data) < 36 {
		return Message{}, fieldError("message", len(data), ErrTruncated)
	}
	if len(data) > 100 {
		return Message{}, fieldError("message.length", 100, ErrRange)
	}
	var m Message
	if err := m.Header.UnmarshalBinary(data[:8]); err != nil {
		return Message{}, err
	}
	if err := supportedHeader(m.Header); err != nil {
		return Message{}, err
	}
	n := int(m.Header.DataLength)
	if n < 28 {
		return Message{}, fieldError("header.dataLength", 6, ErrMalformed)
	}
	if n > len(data)-8 {
		return Message{}, fieldError("common", 8, ErrTruncated)
	}
	c, _, err := DecodeCommonData(data[8:8+n], m.Header.OptionFlags)
	if err != nil {
		return Message{}, rebaseError(err, 8)
	}
	m.Common = c
	pos := 8 + n
	if m.Header.OptionFlags&OptionFree != 0 {
		f, consumed, err := DecodeFreeArea(data[pos:])
		if err != nil {
			return Message{}, rebaseError(err, pos)
		}
		m.Free = &f
		m.freeBase = pos
		m.freeAnchored = true
		pos += consumed
	}
	if pos != len(data) {
		return Message{}, fieldError("message.trailing", pos, ErrMalformed)
	}
	return m, nil
}

// MarshalBinary encodes edited fields into fresh storage. Unknown free data is
// not silently moved when earlier common data changes size. Explicitly rebuilding
// Free as a new FreeArea is the opt-in placement boundary for opaque bytes.
func (m Message) MarshalBinary() ([]byte, error) {
	if err := supportedHeader(m.Header); err != nil {
		return nil, err
	}
	common, err := m.Common.MarshalBinary()
	if err != nil {
		return nil, rebaseError(err, 8)
	}
	if len(common) > 255 {
		return nil, fieldError("common.length", -1, ErrRange)
	}
	h := m.Header
	h.DataLength = uint8(len(common))
	h.OptionFlags = m.Common.OptionFlags()
	var free []byte
	if m.Free != nil {
		if m.freeAnchored && m.Free.entryCount != 0 && 8+len(common) != m.freeBase {
			return nil, fieldError("free.base", 8+len(common), ErrLayout)
		}
		free, err = m.Free.MarshalBinary()
		if err != nil {
			return nil, rebaseError(err, 8+len(common))
		}
		h.OptionFlags |= OptionFree
	}
	if 8+len(common)+len(free) > 100 {
		return nil, fieldError("message.length", -1, ErrRange)
	}
	header, err := h.MarshalBinary()
	if err != nil {
		return nil, err
	}
	out := make([]byte, 0, len(header)+len(common)+len(free))
	out = append(out, header...)
	out = append(out, common...)
	return append(out, free...), nil
}

// Validate reports semantic issues without rejecting or modifying raw values.
// Structural validity is independently enforced by Decode and MarshalBinary.
func (m Message) Validate() []Issue {
	issues := m.Common.Validate()
	if m.Header.ServiceID != 1 || m.Header.MessageID != 1 || m.Header.Version != 1 {
		issues = append(issues, Issue{Field: "header.identifiers", Message: "expected RC-013 basic version-1 identifiers"})
	}
	return issues
}
