package rc016

import (
	"bytes"
	"encoding"
	"fmt"

	"github.com/hareku/its-forum-go/rc013v1"
)

// BicycleFields contains the RC-013 envelope and chapter-3 applications.
// DataLength and OptionFlags are derived from Common and Applications on encode.
type BicycleFields struct {
	Header       rc013.Header
	Common       rc013.CommonData
	Applications []BicycleApplication
}

// BicycleMessage handles both bicycles and pedestrians. The saved experiment
// mapping is immutable. Unknown decoded application starts are pinned to absolute
// packet offsets; moving them requires explicit reconstruction with NewBicycle.
type BicycleMessage struct {
	BicycleFields
	profile BicycleProfile
}

// BicycleApplication owns its preceding gap and payload. For an unmapped ID,
// Data is authoritative. For a mapped ID exactly one matching typed pointer is
// authoritative; Data is only the original snapshot and does not override it.
// Copying this struct follows ordinary Go slice and pointer sharing rules.
type BicycleApplication struct {
	ServiceID      uint8
	Gap, Data      []byte
	PersonalCommon *PersonalCommon
	Basic          *BicycleBasic
	Extended       *BicycleExtended
	Pedestrian     *Pedestrian
	opaqueStart    int
	opaqueAnchored bool
}

func snapshotBicycleProfile(profile BicycleProfile) (BicycleProfile, error) {
	frozen := BicycleProfile{Applications: make(map[uint8]BicycleApplicationKind, len(profile.Applications))}
	for id, kind := range profile.Applications {
		if kind < ApplicationPersonalCommon || kind > ApplicationPedestrian {
			return BicycleProfile{}, &Error{Field: "bicycle.profile", Offset: -1, Err: ErrUnsupported}
		}
		frozen.Applications[id] = kind
	}
	return frozen, nil
}

// DecodeBicycle exposes all common frames and preserves application bytes without
// assigning experiment-specific service IDs to any personal payload kind.
func DecodeBicycle(data []byte) (BicycleMessage, error) {
	return DecodeBicycleWithProfile(data, BicycleProfile{})
}

// DecodeBicycleWithProfile decodes exactly one 36..100-byte chapter-3 message.
// The profile and retained input are copied. Fixed payload mappings require exact sizes.
func DecodeBicycleWithProfile(data []byte, profile BicycleProfile) (BicycleMessage, error) {
	frozen, err := snapshotBicycleProfile(profile)
	if err != nil {
		return BicycleMessage{}, err
	}
	envelope, err := rc013.Decode(data)
	if err != nil {
		return BicycleMessage{}, err
	}
	m := BicycleMessage{BicycleFields: BicycleFields{Header: envelope.Header, Common: envelope.Common}, profile: frozen}
	if envelope.Free == nil {
		return m, nil
	}
	pos := 8 + int(envelope.Header.DataLength) + 1 + 3*len(envelope.Free.Entries)
	for i, e := range envelope.Free.Entries {
		pos += len(e.Gap)
		a := BicycleApplication{ServiceID: e.ServiceID, Gap: e.Gap, Data: e.Data}
		var dst encoding.BinaryUnmarshaler
		switch frozen.Applications[e.ServiceID] {
		case ApplicationPersonalCommon:
			a.PersonalCommon = &PersonalCommon{}
			dst = a.PersonalCommon
		case ApplicationBicycleBasic:
			a.Basic = &BicycleBasic{}
			dst = a.Basic
		case ApplicationBicycleExtended:
			a.Extended = &BicycleExtended{}
			dst = a.Extended
		case ApplicationPedestrian:
			a.Pedestrian = &Pedestrian{}
			dst = a.Pedestrian
		default:
			a.opaqueStart = pos
			a.opaqueAnchored = true
		}
		if dst != nil {
			if err := dst.UnmarshalBinary(e.Data); err != nil {
				return BicycleMessage{}, fmt.Errorf("bicycle.applications[%d]: %w", i, rebaseError(err, pos))
			}
		}
		m.Applications = append(m.Applications, a)
		pos += len(e.Data)
	}
	return m, nil
}

// NewBicycle constructs a message from independently supplied typed fields.
// It copies all buffers, optional frames, and profile data. Opaque bytes are
// explicitly placed by this construction; no original decoded application anchors remain.
func NewBicycle(fields BicycleFields, profile BicycleProfile) (BicycleMessage, error) {
	frozen, err := snapshotBicycleProfile(profile)
	if err != nil {
		return BicycleMessage{}, err
	}
	commonBytes, err := fields.Common.MarshalBinary()
	if err != nil {
		return BicycleMessage{}, rebaseError(err, 8)
	}
	common, _, err := rc013.DecodeCommonData(commonBytes, fields.Common.OptionFlags())
	if err != nil {
		return BicycleMessage{}, rebaseError(err, 8)
	}
	m := BicycleMessage{BicycleFields: BicycleFields{Header: fields.Header, Common: common}, profile: frozen}
	m.Applications = make([]BicycleApplication, len(fields.Applications))
	for i, a := range fields.Applications {
		a.Gap = bytes.Clone(a.Gap)
		a.Data = bytes.Clone(a.Data)
		if a.PersonalCommon != nil {
			v := *a.PersonalCommon
			a.PersonalCommon = &v
		}
		if a.Basic != nil {
			v := *a.Basic
			a.Basic = &v
		}
		if a.Extended != nil {
			v := *a.Extended
			a.Extended = &v
		}
		if a.Pedestrian != nil {
			v := *a.Pedestrian
			a.Pedestrian = &v
		}
		a.opaqueAnchored = false
		a.opaqueStart = 0
		m.Applications[i] = a
	}
	if _, err := m.MarshalBinary(); err != nil {
		return BicycleMessage{}, err
	}
	return m, nil
}

func (a BicycleApplication) payload(kind BicycleApplicationKind) ([]byte, error) {
	count := 0
	if a.PersonalCommon != nil {
		count++
	}
	if a.Basic != nil {
		count++
	}
	if a.Extended != nil {
		count++
	}
	if a.Pedestrian != nil {
		count++
	}
	if kind == 0 {
		if count != 0 {
			return nil, &Error{Field: "bicycle.application.kind", Offset: -1, Err: ErrMalformed}
		}
		return a.Data, nil
	}
	if count != 1 {
		return nil, &Error{Field: "bicycle.application.kind", Offset: -1, Err: ErrMalformed}
	}
	switch kind {
	case ApplicationPersonalCommon:
		if a.PersonalCommon != nil {
			return a.PersonalCommon.MarshalBinary()
		}
	case ApplicationBicycleBasic:
		if a.Basic != nil {
			return a.Basic.MarshalBinary()
		}
	case ApplicationBicycleExtended:
		if a.Extended != nil {
			return a.Extended.MarshalBinary()
		}
	case ApplicationPedestrian:
		if a.Pedestrian != nil {
			return a.Pedestrian.MarshalBinary()
		}
	}
	return nil, &Error{Field: "bicycle.application.kind", Offset: -1, Err: ErrMalformed}
}

// MarshalBinary derives all lengths, counts, flags, and addresses. Fully known
// applications may move; a retained unknown payload may grow only if its start
// stays fixed. Output always has fresh storage. Public fields remain editable.
func (m BicycleMessage) MarshalBinary() ([]byte, error) {
	if len(m.Applications) > 7 {
		return nil, &Error{Field: "bicycle.applications", Offset: -1, Err: ErrRange}
	}
	common, err := m.Common.MarshalBinary()
	if err != nil {
		return nil, rebaseError(err, 8)
	}
	envelope := rc013.Message{Header: m.Header, Common: m.Common}
	if len(m.Applications) > 0 {
		free := rc013.FreeArea{Entries: make([]rc013.FreeEntry, len(m.Applications))}
		start := 8 + len(common) + 1 + 3*len(m.Applications)
		for i, a := range m.Applications {
			// Bound caller-owned lengths before any offset addition or allocation.
			if len(a.Gap) > 59 {
				return nil, &Error{Field: "bicycle.application.gap", Offset: -1, Err: ErrRange}
			}
			data, err := a.payload(m.profile.Applications[a.ServiceID])
			if err != nil {
				return nil, rebaseError(err, start+len(a.Gap))
			}
			if len(data) < 1 || len(data) > 60 {
				return nil, &Error{Field: "bicycle.application.length", Offset: -1, Err: ErrRange}
			}
			start += len(a.Gap)
			if a.opaqueAnchored && start != a.opaqueStart {
				return nil, &Error{Field: fmt.Sprintf("bicycle.applications[%d]", i), Offset: start, Err: ErrLayout}
			}
			free.Entries[i] = rc013.FreeEntry{ServiceID: a.ServiceID, Gap: a.Gap, Data: data}
			start += len(data)
		}
		envelope.Free = &free
	}
	return envelope.MarshalBinary()
}

// Validate reports semantic concerns without changing wire values or layout.
func (m BicycleMessage) Validate() []Issue {
	issues := (rc013.Message{Header: m.Header, Common: m.Common}).Validate()
	add := func(i int, field, message string, bad bool) {
		if bad {
			issues = append(issues, Issue{Field: fmt.Sprintf("bicycle.applications[%d].%s", i, field), Message: message})
		}
	}
	for i, a := range m.Applications {
		if p := a.PersonalCommon; p != nil {
			add(i, "level", "unassigned personal level", p.Level < 1 || p.Level > 5)
		}
		if p := a.Basic; p != nil {
			add(i, "assistType", "reserved assist type", p.AssistType > 2)
			add(i, "pedaling", "reserved pedaling state", p.Pedaling == 3)
		}
		if p := a.Extended; p != nil {
			add(i, "rearLight", "reserved rear light state", p.RearLight == 3)
			add(i, "driveUnitStatus", "reserved drive unit status", p.DriveUnitStatus == 3)
			add(i, "maintenance", "reserved maintenance state", p.Maintenance == 3)
			add(i, "reserved", "reserved bits are nonzero", p.Reserved != 0)
		}
		if p := a.Pedestrian; p != nil {
			add(i, "shoeAttribute", "experiment-defined shoe attribute", p.ShoeAttribute < 1 || p.ShoeAttribute > 3)
			add(i, "reserved", "reserved bits are nonzero", p.Reserved != 0)
		}
	}
	return issues
}
