package rc013

import "bytes"

// FreeEntry owns application bytes and the gap immediately before them.
// Addresses and lengths are derived from entries, preserving explicit gap bytes.
type FreeEntry struct {
	ServiceID uint8
	Gap, Data []byte
	origin    int
}

// FreeArea is a present free application area with one through seven entries.
// Decoded entries retain private position anchors. Explicit reconstruction as
// FreeArea{Entries: ...} discards those anchors and opts into a new placement.
type FreeArea struct {
	Entries    []FreeEntry
	entryCount int
}

// DecodeFreeArea consumes the header and the last payload's end, leaving bytes
// belonging to a containing message unread. It does not impose a 100-byte packet cap.
func DecodeFreeArea(data []byte) (FreeArea, int, error) {
	if len(data) < 1 {
		return FreeArea{}, 0, fieldError("free.header", 0, ErrTruncated)
	}
	headerLen, count := int(data[0]>>3), int(data[0]&7)
	if count == 0 || headerLen != 1+3*count {
		return FreeArea{}, 0, fieldError("free.header", 0, ErrMalformed)
	}
	if len(data) < headerLen {
		return FreeArea{}, 0, fieldError("free.header", len(data), ErrTruncated)
	}
	// Validate every span before allocating any entry payload.
	end := 0
	for i := 0; i < count; i++ {
		p := 1 + 3*i
		start, size := int(data[p+1]), int(data[p+2])
		if start > 59 || size < 1 || size > 60 || start < end {
			return FreeArea{}, 0, fieldError("free.entry", p, ErrMalformed)
		}
		if start+size > len(data)-headerLen {
			return FreeArea{}, 0, fieldError("free.payload", headerLen+start, ErrTruncated)
		}
		end = start + size
	}
	f := FreeArea{Entries: make([]FreeEntry, count), entryCount: count}
	previous := 0
	for i := range f.Entries {
		p := 1 + 3*i
		start, size := int(data[p+1]), int(data[p+2])
		f.Entries[i] = FreeEntry{ServiceID: data[p], Gap: bytes.Clone(data[headerLen+previous : headerLen+start]), Data: bytes.Clone(data[headerLen+start : headerLen+start+size]), origin: headerLen + start}
		previous = start + size
	}
	return f, headerLen + end, nil
}

// MarshalBinary derives descriptors and refuses to move previously decoded
// opaque payloads. A final payload may grow while its start remains fixed.
func (f FreeArea) MarshalBinary() ([]byte, error) {
	count := len(f.Entries)
	if count < 1 || count > 7 {
		return nil, fieldError("free.entries", -1, ErrRange)
	}
	headerLen := 1 + 3*count
	if f.entryCount != 0 && f.entryCount != count {
		return nil, fieldError("free.entries", -1, ErrLayout)
	}
	end := 0
	for _, e := range f.Entries {
		if len(e.Gap) > 59-end {
			return nil, fieldError("free.address", -1, ErrRange)
		}
		start := end + len(e.Gap)
		if len(e.Data) < 1 || len(e.Data) > 60 {
			return nil, fieldError("free.length", -1, ErrRange)
		}
		if f.entryCount != 0 && headerLen+start != e.origin {
			return nil, fieldError("free.payload", headerLen+start, ErrLayout)
		}
		end = start + len(e.Data)
	}
	out := make([]byte, headerLen+end)
	out[0] = byte(headerLen<<3 | count)
	pos := 0
	for i, e := range f.Entries {
		copy(out[headerLen+pos:], e.Gap)
		pos += len(e.Gap)
		p := 1 + 3*i
		out[p] = e.ServiceID
		out[p+1] = byte(pos)
		out[p+2] = byte(len(e.Data))
		copy(out[headerLen+pos:], e.Data)
		pos += len(e.Data)
	}
	return out, nil
}
