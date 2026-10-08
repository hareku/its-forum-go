package rc016

import (
	"bytes"
	"encoding"
	"errors"
	"fmt"

	"github.com/hareku/its-forum-go/rc013v1"
)

// Fields contains the RC-013 envelope and chapter-3 applications.
// DataLength and OptionFlags are derived from Common and Applications on encode.
type Fields struct {
	Header       rc013.Header
	Common       rc013.CommonData
	Applications []Application
}

// Message handles both bicycles and pedestrians. The saved experiment
// mapping is immutable. Unknown decoded application starts are pinned to absolute
// packet offsets; moving them requires explicit reconstruction with NewMessage.
type Message struct {
	Fields
	profile Profile
}

// Application owns its preceding gap and payload. For an unmapped ID,
// Data is authoritative. For a mapped ID exactly one matching typed pointer is
// authoritative; Data is only the original snapshot and does not override it.
// Copying this struct follows ordinary Go slice and pointer sharing rules.
type Application struct {
	ServiceID      uint8
	Gap, Data      []byte
	Bicycle        *BicycleData
	Pedestrian     *PedestrianData
	opaqueStart    int
	opaqueAnchored bool
}

// Decode exposes all common frames and preserves application bytes without
// assigning experiment-specific service IDs to any personal payload kind.
func Decode(data []byte) (Message, error) {
	return DecodeWithProfile(data, Profile{})
}

// DecodeWithProfile decodes exactly one 36..100-byte chapter-3 message.
// The profile and retained input are copied. Fixed payload mappings require exact sizes.
func DecodeWithProfile(data []byte, profile Profile) (Message, error) {
	frozen, err := snapshotProfile(profile)
	if err != nil {
		return Message{}, err
	}
	envelope, err := rc013.Decode(data)
	if err != nil {
		return Message{}, err
	}
	m := Message{Fields: Fields{Header: envelope.Header, Common: envelope.Common}, profile: frozen}
	if envelope.Free == nil {
		return m, nil
	}
	pos := 8 + int(envelope.Header.DataLength) + 1 + 3*len(envelope.Free.Entries)
	for i, e := range envelope.Free.Entries {
		pos += len(e.Gap)
		a := Application{ServiceID: e.ServiceID, Gap: e.Gap, Data: e.Data}
		var dst encoding.BinaryUnmarshaler
		switch frozen.Applications[e.ServiceID] {
		case ApplicationBicycle:
			a.Bicycle = &BicycleData{}
			dst = a.Bicycle
		case ApplicationPedestrian:
			a.Pedestrian = &PedestrianData{}
			dst = a.Pedestrian
		default:
			a.opaqueStart = pos
			a.opaqueAnchored = true
		}
		if dst != nil {
			if err := dst.UnmarshalBinary(e.Data); err != nil {
				return Message{}, applicationError(err, i, pos)
			}
		}
		m.Applications = append(m.Applications, a)
		pos += len(e.Data)
	}
	return m, nil
}

// NewMessage constructs a message from independently supplied typed fields.
// It copies all buffers, optional frames, and profile data. Opaque bytes are
// explicitly placed by this construction; no original decoded application anchors remain.
func NewMessage(fields Fields, profile Profile) (Message, error) {
	frozen, err := snapshotProfile(profile)
	if err != nil {
		return Message{}, err
	}
	if len(fields.Applications) > 7 {
		return Message{}, &Error{Field: "message.applications", Offset: -1, Err: ErrRange}
	}
	// Rebuilding public common fields deliberately clears decoded tail anchors.
	c := fields.Common
	rebuilt := rc013.CommonData{Time: c.Time, Position: c.Position, VehicleState: c.VehicleState,
		VehicleAttributes: c.VehicleAttributes, PositionOptional: c.PositionOptional,
		GPSStatusOptional: c.GPSStatusOptional, PositionAcquisitionOptional: c.PositionAcquisitionOptional,
		VehicleStateOptional: c.VehicleStateOptional, Intersection: c.Intersection, Extension: c.Extension,
		ExtendedOptions: c.ExtendedOptions, OpaqueTail: c.OpaqueTail}
	commonBytes, err := rebuilt.MarshalBinary()
	if err != nil {
		return Message{}, rebaseError(err, 8)
	}
	common, _, err := rc013.DecodeCommonData(commonBytes, fields.Common.OptionFlags())
	if err != nil {
		return Message{}, rebaseError(err, 8)
	}
	m := Message{Fields: Fields{Header: fields.Header, Common: common}, profile: frozen}
	m.Applications = make([]Application, len(fields.Applications))
	for i, a := range fields.Applications {
		a.Gap = bytes.Clone(a.Gap)
		a.Data = bytes.Clone(a.Data)
		if a.Bicycle != nil {
			v := *a.Bicycle
			a.Bicycle = &v
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
		return Message{}, err
	}
	return m, nil
}

func (a Application) payload(kind ApplicationKind) ([]byte, error) {
	switch kind {
	case 0:
		if a.Bicycle == nil && a.Pedestrian == nil {
			return a.Data, nil
		}
	case ApplicationBicycle:
		if a.Bicycle != nil && a.Pedestrian == nil {
			return a.Bicycle.MarshalBinary()
		}
	case ApplicationPedestrian:
		if a.Pedestrian != nil && a.Bicycle == nil {
			return a.Pedestrian.MarshalBinary()
		}
	}
	return nil, &Error{Field: "kind", Offset: -1, Err: ErrMalformed}
}

// applicationError retains both the packet coordinate and application field path.
func applicationError(err error, index, base int) error {
	var located *Error
	if errors.As(rebaseError(err, base), &located) {
		next := *located
		next.Field = fmt.Sprintf("message.applications[%d].%s", index, located.Field)
		return &next
	}
	return fmt.Errorf("message.applications[%d]: %w", index, err)
}

// MarshalBinary derives all lengths, counts, flags, and addresses. Fully known
// applications may move; a retained unknown payload may grow only if its start
// stays fixed. Output always has fresh storage. Public fields remain editable.
func (m Message) MarshalBinary() ([]byte, error) {
	if len(m.Applications) > 7 {
		return nil, &Error{Field: "message.applications", Offset: -1, Err: ErrRange}
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
				return nil, &Error{Field: fmt.Sprintf("message.applications[%d].gap", i), Offset: -1, Err: ErrRange}
			}
			data, err := a.payload(m.profile.Applications[a.ServiceID])
			if err != nil {
				return nil, applicationError(err, i, start+len(a.Gap))
			}
			if len(data) < 1 || len(data) > 60 {
				return nil, &Error{Field: fmt.Sprintf("message.applications[%d].length", i), Offset: -1, Err: ErrRange}
			}
			start += len(a.Gap)
			if a.opaqueAnchored && start != a.opaqueStart {
				return nil, &Error{Field: fmt.Sprintf("message.applications[%d]", i), Offset: start, Err: ErrLayout}
			}
			free.Entries[i] = rc013.FreeEntry{ServiceID: a.ServiceID, Gap: a.Gap, Data: data}
			start += len(data)
		}
		envelope.Free = &free
	}
	return envelope.MarshalBinary()
}
