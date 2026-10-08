package rc016

import (
	"bytes"
	"github.com/hareku/its-forum-go/rc016v1/internal/bitio"
)

// SensorAttributeLayout selects an explicitly supplied experimental layout.
type SensorAttributeLayout uint8

const (
	SensorOpaque SensorAttributeLayout = iota
	SensorExample
)

// SensorStatusField locates an experimental operating state within one sensor.
type SensorStatusField struct {
	BitOffset, Width int
	ActiveValue      uint32
}

// SensorProfile describes one sensor, independently of group ordering.
type SensorProfile struct {
	Layout SensorAttributeLayout
	Status *SensorStatusField
}

// RoadsideProfile describes experiment-specific attributes and object extensions.
// GroupOrder lists sensor indices, including inactive sensors. A nil list is unresolved.
type RoadsideProfile struct {
	Sensors          []SensorProfile
	GroupOrder       []int
	ObjectExtensions ObjectExtensionProfile
}

// ObjectExtensionProfile maps experimental service IDs to explicit DE layouts.
type ObjectExtensionProfile struct {
	Layouts map[uint8]ObjectExtensionLayout
}

// ObjectExtensionLayout lists field presence and order. Partial layouts retain a suffix.
type ObjectExtensionLayout struct {
	Fields   []ObjectExtensionField
	Complete bool
}

// ObjectExtensionField identifies a source-defined data element, never a wire service ID.
type ObjectExtensionField uint8

const (
	ObjectLevel ObjectExtensionField = iota
	ObjectFusion
	ObjectFusionSources
	ObjectMonitoringData
	ObjectAssistType
	ObjectBicycleType
	ObjectAssistState
	ObjectPedaling
	ObjectDrivePower
	ObjectCollision
	ObjectMainGear
	ObjectMainMaxGear
	ObjectSubGear
	ObjectSubMaxGear
	ObjectTireCircumference
	ObjectCadence
	ObjectGearRatio
	ObjectRiderTorque
	ObjectMotorTorque
	ObjectMaxAssistPower
	ObjectAssistPower
	ObjectHumanPower
	ObjectMaxBattery
	ObjectBattery
	ObjectRearLight
	ObjectDriveUnitStatus
	ObjectMaintenance
	ObjectExtendedReserved
	ObjectShoeAttribute
	ObjectSteps
	ObjectMotion
	ObjectPedestrianReserved
)

var roadsideObjectFieldWidths = [...]int{3, 2, 3, 32, 4, 4, 2, 2, 8, 4, 5, 5, 5, 5, 8, 8, 10, 8, 8, 8, 8, 8, 8, 8, 2, 2, 2, 4, 6, 14, 2, 18}

// ObjectExtensionFields stores exact encoded integers. Presence comes from the profile.
type ObjectExtensionFields struct {
	Level, Fusion, FusionSources, MonitoringData                                           uint32
	AssistType, BicycleType, AssistState, Pedaling, DrivePower, Collision                  uint32
	MainGear, MainMaxGear, SubGear, SubMaxGear, TireCircumference, Cadence, GearRatio      uint32
	RiderTorque, MotorTorque, MaxAssistPower, AssistPower, HumanPower, MaxBattery, Battery uint32
	RearLight, DriveUnitStatus, Maintenance, ExtendedReserved                              uint32
	ShoeAttribute, Steps, Motion, PedestrianReserved                                       uint32
}

func (f *ObjectExtensionFields) values() []*uint32 {
	return []*uint32{&f.Level, &f.Fusion, &f.FusionSources, &f.MonitoringData, &f.AssistType, &f.BicycleType, &f.AssistState, &f.Pedaling, &f.DrivePower, &f.Collision, &f.MainGear, &f.MainMaxGear, &f.SubGear, &f.SubMaxGear, &f.TireCircumference, &f.Cadence, &f.GearRatio, &f.RiderTorque, &f.MotorTorque, &f.MaxAssistPower, &f.AssistPower, &f.HumanPower, &f.MaxBattery, &f.Battery, &f.RearLight, &f.DriveUnitStatus, &f.Maintenance, &f.ExtendedReserved, &f.ShoeAttribute, &f.Steps, &f.Motion, &f.PedestrianReserved}
}
func freezeRoadsideProfile(p RoadsideProfile) (RoadsideProfile, error) {
	q := RoadsideProfile{Sensors: append([]SensorProfile(nil), p.Sensors...), ObjectExtensions: ObjectExtensionProfile{Layouts: make(map[uint8]ObjectExtensionLayout)}}
	if p.GroupOrder != nil {
		q.GroupOrder = append([]int{}, p.GroupOrder...)
	}
	for i, s := range q.Sensors {
		if s.Layout > SensorExample || (s.Layout == SensorExample && s.Status != nil) {
			return q, roadsideError("profile.sensor", -1, ErrMalformed)
		}
		if s.Status != nil {
			v := *s.Status
			if v.BitOffset < 0 || v.BitOffset > 2040 || v.Width < 1 || v.Width > 32 || uint64(v.ActiveValue) >= uint64(1)<<v.Width {
				return q, roadsideError("profile.status", -1, ErrRange)
			}
			q.Sensors[i].Status = &v
		}
	}
	seen := map[int]bool{}
	for _, i := range q.GroupOrder {
		if i < 0 || i >= len(q.Sensors) || seen[i] {
			return q, roadsideError("profile.order", -1, ErrMalformed)
		}
		seen[i] = true
	}
	if q.GroupOrder != nil && len(q.GroupOrder) != len(q.Sensors) {
		return q, roadsideError("profile.order", -1, ErrMalformed)
	}
	for id, l := range p.ObjectExtensions.Layouts {
		seenFields := map[ObjectExtensionField]bool{}
		for _, f := range l.Fields {
			if int(f) >= len(roadsideObjectFieldWidths) || seenFields[f] {
				return q, roadsideError("profile.extension", -1, ErrMalformed)
			}
			seenFields[f] = true
		}
		l.Fields = append([]ObjectExtensionField(nil), l.Fields...)
		q.ObjectExtensions.Layouts[id] = l
	}
	return q, nil
}
func roadsideExtensionBits(l ObjectExtensionLayout) int {
	n := 0
	for _, f := range l.Fields {
		n += roadsideObjectFieldWidths[f]
	}
	return n
}
func decodeRoadsideObjectFields(data []byte, l ObjectExtensionLayout) (*ObjectExtensionFields, error) {
	n := (roadsideExtensionBits(l) + 7) / 8
	if len(data) < n || (l.Complete && len(data) != n) {
		return nil, roadsideError("extension.length", -1, ErrMalformed)
	}
	f := new(ObjectExtensionFields)
	v := f.values()
	off := 0
	for _, field := range l.Fields {
		x, e := bitio.Read(data, off, roadsideObjectFieldWidths[field])
		if e != nil {
			return nil, e
		}
		*v[field] = uint32(x)
		off += roadsideObjectFieldWidths[field]
	}
	return f, nil
}
func encodeRoadsideObjectFields(e RoadsideObjectExtension, l ObjectExtensionLayout) ([]byte, error) {
	if e.Fields == nil {
		return nil, roadsideError("extension.fields", -1, ErrMalformed)
	}
	n := (roadsideExtensionBits(l) + 7) / 8
	data := bytes.Clone(e.Data)
	if data == nil {
		data = make([]byte, n)
	}
	if len(data) < n || (l.Complete && len(data) != n) {
		return nil, roadsideError("extension.length", -1, ErrMalformed)
	}
	off := 0
	v := e.Fields.values()
	present := make(map[ObjectExtensionField]bool, len(l.Fields))
	for _, field := range l.Fields {
		present[field] = true
	}
	for field, value := range v {
		if !present[ObjectExtensionField(field)] && *value != 0 {
			return nil, roadsideError("extension.absent_field", -1, ErrMalformed)
		}
	}
	for _, f := range l.Fields {
		if uint64(*v[f]) >= uint64(1)<<roadsideObjectFieldWidths[f] {
			return nil, roadsideError("extension.value", -1, ErrRange)
		}
		if err := bitio.Write(data, off, roadsideObjectFieldWidths[f], uint64(*v[f])); err != nil {
			return nil, err
		}
		off += roadsideObjectFieldWidths[f]
	}
	return data, nil
}
