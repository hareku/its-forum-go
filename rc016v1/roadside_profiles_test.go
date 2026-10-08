package rc016

import (
	"bytes"
	"errors"
	"github.com/hareku/its-forum-go/rc013v1"
	"testing"
)

func TestRoadsideProfileGolden(t *testing.T) {
	b := roadsideHex(t, roadsideProfileGolden)
	p := roadsideProfile()
	m, e := DecodeRoadsideWithProfile(b, p)
	if e != nil {
		t.Fatal(e)
	}
	if !m.ObjectsDecoded() || len(m.Groups) != 1 || m.Groups[0].SensorIndex != 0 || m.Options[0].Sensors[1].Example.Status != 1 {
		t.Fatal("topology")
	}
	x := m.Groups[0].Objects[0].Extensions[0]
	if x.Fields.Level != 5 || x.Fields.Fusion != 2 || x.Fields.FusionSources != 5 || x.Fields.AssistType != 2 {
		t.Fatal("fields")
	}
	built, e := NewRoadside(roadsideProfileFixture(), p)
	if e != nil {
		t.Fatal(e)
	}
	out, e := built.MarshalBinary()
	if e != nil || !bytes.Equal(out, b) {
		t.Fatalf("profile construct %x %v", out, e)
	}
	p.GroupOrder[0] = 0
	p.Sensors[0].Layout = SensorOpaque
	p.ObjectExtensions.Layouts[0x91].Fields[0] = ObjectCollision
	delete(p.ObjectExtensions.Layouts, 0x91)
	for _, model := range []RoadsideMessage{m, built} {
		out, e := model.MarshalBinary()
		if e != nil || !bytes.Equal(out, b) {
			t.Fatalf("profile alias %v", e)
		}
	}
	m.Groups[0].Objects[0].Extensions[0].Fields.Fusion = 1
	out, e = m.MarshalBinary()
	expected := bytes.Clone(b)
	expected[len(expected)-2] = 0xad
	if e != nil || !bytes.Equal(out, expected) {
		t.Fatalf("fusion edit %x %v", out, e)
	}
	for n := 0; n < len(b); n++ {
		if _, e := DecodeRoadsideWithProfile(b[:n], roadsideProfile()); e == nil {
			t.Fatal("prefix", n)
		}
	}
}
func TestRoadsideUnresolvedAndSensorOrder(t *testing.T) {
	b := roadsideHex(t, roadsideProfileGolden)
	m, e := DecodeRoadside(b)
	if e != nil {
		t.Fatal(e)
	}
	if m.ObjectsDecoded() || len(m.Groups) != 0 || len(m.Options[0].Sensors) != 2 {
		t.Fatal("unresolved")
	}
	out, e := m.MarshalBinary()
	if e != nil || !bytes.Equal(out, b) {
		t.Fatal(e)
	}
	m.Header.Counter++
	out, e = m.MarshalBinary()
	if e != nil || out[1] != 43 || !bytes.Equal(out[2:], b[2:]) {
		t.Fatal("header edit")
	}
	m.Options[0].Sensors[0].Attributes = append(m.Options[0].Sensors[0].Attributes, 0x12)
	if _, e := m.MarshalBinary(); !errors.Is(e, ErrLayout) {
		t.Fatal("suffix movement", e)
	}
	f := roadsideProfileFixture()
	p := roadsideProfile()
	f.Options[0].Sensors[1].Example.Status = 0
	g := f.Groups[0]
	g.SensorIndex = 1
	g.Objects = append([]RoadsideObject(nil), g.Objects...)
	g.Objects[0].ID = 7
	f.Groups = append([]RoadsideObjectGroup{g}, f.Groups...)
	m, e = NewRoadside(f, p)
	if e != nil {
		t.Fatal(e)
	}
	if len(m.Groups) != 2 || m.Groups[0].SensorIndex != 1 || m.Groups[0].Objects[0].ID != 7 || m.Groups[1].Objects[0].ID != 0x11223344 {
		t.Fatal("group order")
	}
	f = roadsideProfileFixture()
	f.Options[0].Sensors[0].Example.Status = 3
	f.Groups = nil
	m, e = NewRoadside(f, p)
	if e != nil || !m.ObjectsDecoded() || len(m.Groups) != 0 {
		t.Fatal("all inactive", e)
	}
	f = roadsideFixture()
	f.Options = []RoadsideOption{{Index: 0, Sensors: []RoadsideSensor{}}}
	f.Groups = nil
	p = RoadsideProfile{GroupOrder: []int{}}
	m, e = NewRoadside(f, p)
	if e != nil || !m.ObjectsDecoded() || len(m.Validate()) == 0 {
		t.Fatal("zero sensors", e)
	}
}
func TestRoadsideExtensionProfiles(t *testing.T) {
	b := roadsideHex(t, roadsideProfileGolden)
	p := roadsideProfile()
	p.ObjectExtensions.Layouts[0x91] = ObjectExtensionLayout{Fields: []ObjectExtensionField{ObjectLevel, ObjectFusion, ObjectFusionSources}}
	m, e := DecodeRoadsideWithProfile(b, p)
	if e != nil {
		t.Fatal(e)
	}
	out, e := m.MarshalBinary()
	if e != nil || !bytes.Equal(out, b) {
		t.Fatal("partial", e)
	}
	m.Groups[0].Objects[0].Extensions[0].Fields.Level = 4
	out, e = m.MarshalBinary()
	if e != nil || out[len(out)-2] != 0x95 || out[len(out)-1] != 0x2b {
		t.Fatal("partial mask")
	}
	m.Options[0].Sensors[0].Example.Extra = []byte{1}
	if _, e := m.MarshalBinary(); !errors.Is(e, ErrLayout) {
		t.Fatal("partial absolute anchor", e)
	}
	p = roadsideProfile()
	p.ObjectExtensions.Layouts[0x91] = ObjectExtensionLayout{Fields: []ObjectExtensionField{ObjectLevel, ObjectFusion, ObjectFusionSources}, Complete: true}
	if _, e := DecodeRoadsideWithProfile(b, p); e == nil {
		t.Fatal("complete mismatch")
	}
	f := roadsideProfileFixture()
	x := &f.Groups[0].Objects[0].Extensions[0]
	x.Data = nil
	x.Fields.AssistType = 0
	m, e = NewRoadside(f, p)
	if e != nil {
		t.Fatal(e)
	}
	out, _ = m.MarshalBinary()
	if len(out) != 74 || out[len(out)-1] != 0xb5 || out[13] != 58 || out[len(out)-2] != 1 {
		t.Fatalf("new presence %x", out)
	}
	p.ObjectExtensions.Layouts[0x91] = ObjectExtensionLayout{Fields: []ObjectExtensionField{ObjectAssistType, ObjectLevel, ObjectFusion, ObjectFusionSources}, Complete: true}
	f = roadsideProfileFixture()
	f.Groups[0].Objects[0].Extensions[0].Data = []byte{0, 0xb}
	m, e = NewRoadside(f, p)
	if e != nil {
		t.Fatal(e)
	}
	out, _ = m.MarshalBinary()
	if !bytes.Equal(out[len(out)-2:], []byte{0x2b, 0x5b}) {
		t.Fatalf("explicit order %x", out)
	}
	p = roadsideProfile()
	f = roadsideProfileFixture()
	f.Groups[0].Objects[0].Extensions = append(f.Groups[0].Objects[0].Extensions, RoadsideObjectExtension{ServiceID: 0x91, Fields: &ObjectExtensionFields{Level: 1}})
	m, e = NewRoadside(f, p)
	if e != nil {
		t.Fatal("known resize", e)
	}
	m.Groups[0].Objects[0].Extensions = m.Groups[0].Objects[0].Extensions[1:]
	out, e = m.MarshalBinary()
	if e != nil || len(out) != 75 {
		t.Fatal("known remove and relocation", e)
	}
}
func TestRoadsideProfileBounds(t *testing.T) {
	for _, p := range []RoadsideProfile{{GroupOrder: []int{0}}, {Sensors: []SensorProfile{{Layout: 9}}}, {Sensors: []SensorProfile{{Status: &SensorStatusField{Width: 33}}}}, {ObjectExtensions: ObjectExtensionProfile{Layouts: map[uint8]ObjectExtensionLayout{1: {Fields: []ObjectExtensionField{ObjectLevel, ObjectLevel}}}}}} {
		if _, e := DecodeRoadsideWithProfile(roadsideHex(t, roadsideGolden), p); e == nil {
			t.Fatal("invalid profile")
		}
	}
	p := roadsideProfile()
	p.Sensors = p.Sensors[:1]
	p.GroupOrder = []int{0}
	if _, e := DecodeRoadsideWithProfile(roadsideHex(t, roadsideProfileGolden), p); e == nil {
		t.Fatal("sensor mismatch")
	}
	p = roadsideProfile()
	p.Sensors = []SensorProfile{{Status: &SensorStatusField{BitOffset: 25, Width: 15}}, {Status: &SensorStatusField{BitOffset: 25, Width: 15}}}
	m, e := DecodeRoadsideWithProfile(roadsideHex(t, roadsideProfileGolden), p)
	if e != nil || len(m.Groups) != 1 {
		t.Fatal("explicit bit selector", e)
	}
	p.Sensors[0].Status.BitOffset = 100
	out, e := m.MarshalBinary()
	if e != nil || !bytes.Equal(out, roadsideHex(t, roadsideProfileGolden)) {
		t.Fatal("selector alias", e)
	}
}

func TestRoadsideExtensionEveryField(t *testing.T) {
	// Independent specification widths: RC-016 table 4-8 and chapter 3 DE tables.
	// Named accessors deliberately do not use the production field-index mapping.
	cases := []struct {
		name  string
		field ObjectExtensionField
		width int
		set   func(*ObjectExtensionFields, uint32)
		get   func(*ObjectExtensionFields) uint32
	}{
		{"Level", ObjectLevel, 3, func(f *ObjectExtensionFields, v uint32) { f.Level = v }, func(f *ObjectExtensionFields) uint32 { return f.Level }},
		{"Fusion", ObjectFusion, 2, func(f *ObjectExtensionFields, v uint32) { f.Fusion = v }, func(f *ObjectExtensionFields) uint32 { return f.Fusion }},
		{"FusionSources", ObjectFusionSources, 3, func(f *ObjectExtensionFields, v uint32) { f.FusionSources = v }, func(f *ObjectExtensionFields) uint32 { return f.FusionSources }},
		{"MonitoringData", ObjectMonitoringData, 32, func(f *ObjectExtensionFields, v uint32) { f.MonitoringData = v }, func(f *ObjectExtensionFields) uint32 { return f.MonitoringData }},
		{"AssistType", ObjectAssistType, 4, func(f *ObjectExtensionFields, v uint32) { f.AssistType = v }, func(f *ObjectExtensionFields) uint32 { return f.AssistType }},
		{"BicycleType", ObjectBicycleType, 4, func(f *ObjectExtensionFields, v uint32) { f.BicycleType = v }, func(f *ObjectExtensionFields) uint32 { return f.BicycleType }},
		{"AssistState", ObjectAssistState, 2, func(f *ObjectExtensionFields, v uint32) { f.AssistState = v }, func(f *ObjectExtensionFields) uint32 { return f.AssistState }},
		{"Pedaling", ObjectPedaling, 2, func(f *ObjectExtensionFields, v uint32) { f.Pedaling = v }, func(f *ObjectExtensionFields) uint32 { return f.Pedaling }},
		{"DrivePower", ObjectDrivePower, 8, func(f *ObjectExtensionFields, v uint32) { f.DrivePower = v }, func(f *ObjectExtensionFields) uint32 { return f.DrivePower }},
		{"Collision", ObjectCollision, 4, func(f *ObjectExtensionFields, v uint32) { f.Collision = v }, func(f *ObjectExtensionFields) uint32 { return f.Collision }},
		{"MainGear", ObjectMainGear, 5, func(f *ObjectExtensionFields, v uint32) { f.MainGear = v }, func(f *ObjectExtensionFields) uint32 { return f.MainGear }},
		{"MainMaxGear", ObjectMainMaxGear, 5, func(f *ObjectExtensionFields, v uint32) { f.MainMaxGear = v }, func(f *ObjectExtensionFields) uint32 { return f.MainMaxGear }},
		{"SubGear", ObjectSubGear, 5, func(f *ObjectExtensionFields, v uint32) { f.SubGear = v }, func(f *ObjectExtensionFields) uint32 { return f.SubGear }},
		{"SubMaxGear", ObjectSubMaxGear, 5, func(f *ObjectExtensionFields, v uint32) { f.SubMaxGear = v }, func(f *ObjectExtensionFields) uint32 { return f.SubMaxGear }},
		{"TireCircumference", ObjectTireCircumference, 8, func(f *ObjectExtensionFields, v uint32) { f.TireCircumference = v }, func(f *ObjectExtensionFields) uint32 { return f.TireCircumference }},
		{"Cadence", ObjectCadence, 8, func(f *ObjectExtensionFields, v uint32) { f.Cadence = v }, func(f *ObjectExtensionFields) uint32 { return f.Cadence }},
		{"GearRatio", ObjectGearRatio, 10, func(f *ObjectExtensionFields, v uint32) { f.GearRatio = v }, func(f *ObjectExtensionFields) uint32 { return f.GearRatio }},
		{"RiderTorque", ObjectRiderTorque, 8, func(f *ObjectExtensionFields, v uint32) { f.RiderTorque = v }, func(f *ObjectExtensionFields) uint32 { return f.RiderTorque }},
		{"MotorTorque", ObjectMotorTorque, 8, func(f *ObjectExtensionFields, v uint32) { f.MotorTorque = v }, func(f *ObjectExtensionFields) uint32 { return f.MotorTorque }},
		{"MaxAssistPower", ObjectMaxAssistPower, 8, func(f *ObjectExtensionFields, v uint32) { f.MaxAssistPower = v }, func(f *ObjectExtensionFields) uint32 { return f.MaxAssistPower }},
		{"AssistPower", ObjectAssistPower, 8, func(f *ObjectExtensionFields, v uint32) { f.AssistPower = v }, func(f *ObjectExtensionFields) uint32 { return f.AssistPower }},
		{"HumanPower", ObjectHumanPower, 8, func(f *ObjectExtensionFields, v uint32) { f.HumanPower = v }, func(f *ObjectExtensionFields) uint32 { return f.HumanPower }},
		{"MaxBattery", ObjectMaxBattery, 8, func(f *ObjectExtensionFields, v uint32) { f.MaxBattery = v }, func(f *ObjectExtensionFields) uint32 { return f.MaxBattery }},
		{"Battery", ObjectBattery, 8, func(f *ObjectExtensionFields, v uint32) { f.Battery = v }, func(f *ObjectExtensionFields) uint32 { return f.Battery }},
		{"RearLight", ObjectRearLight, 2, func(f *ObjectExtensionFields, v uint32) { f.RearLight = v }, func(f *ObjectExtensionFields) uint32 { return f.RearLight }},
		{"DriveUnitStatus", ObjectDriveUnitStatus, 2, func(f *ObjectExtensionFields, v uint32) { f.DriveUnitStatus = v }, func(f *ObjectExtensionFields) uint32 { return f.DriveUnitStatus }},
		{"Maintenance", ObjectMaintenance, 2, func(f *ObjectExtensionFields, v uint32) { f.Maintenance = v }, func(f *ObjectExtensionFields) uint32 { return f.Maintenance }},
		{"ExtendedReserved", ObjectExtendedReserved, 4, func(f *ObjectExtensionFields, v uint32) { f.ExtendedReserved = v }, func(f *ObjectExtensionFields) uint32 { return f.ExtendedReserved }},
		{"ShoeAttribute", ObjectShoeAttribute, 6, func(f *ObjectExtensionFields, v uint32) { f.ShoeAttribute = v }, func(f *ObjectExtensionFields) uint32 { return f.ShoeAttribute }},
		{"Steps", ObjectSteps, 14, func(f *ObjectExtensionFields, v uint32) { f.Steps = v }, func(f *ObjectExtensionFields) uint32 { return f.Steps }},
		{"Motion", ObjectMotion, 2, func(f *ObjectExtensionFields, v uint32) { f.Motion = v }, func(f *ObjectExtensionFields) uint32 { return f.Motion }},
		{"PedestrianReserved", ObjectPedestrianReserved, 18, func(f *ObjectExtensionFields, v uint32) { f.PedestrianReserved = v }, func(f *ObjectExtensionFields) uint32 { return f.PedestrianReserved }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := RoadsideProfile{ObjectExtensions: ObjectExtensionProfile{Layouts: map[uint8]ObjectExtensionLayout{7: {Fields: []ObjectExtensionField{tc.field}, Complete: true}}}}
			maximum := uint32((uint64(1) << tc.width) - 1)
			for _, value := range []uint32{0, 1, maximum} {
				fields := new(ObjectExtensionFields)
				tc.set(fields, value)
				f := roadsideFixture()
				f.Groups[0].Objects[0].Extensions = []RoadsideObjectExtension{{ServiceID: 7, Fields: fields}}
				m, e := NewRoadside(f, p)
				if e != nil {
					t.Fatal(e)
				}
				got, e := m.MarshalBinary()
				if e != nil {
					t.Fatal(e)
				}
				n := (tc.width + 7) / 8
				packed := uint64(value) << uint(n*8-tc.width)
				payload := make([]byte, n)
				for i := n - 1; i >= 0; i-- {
					payload[i] = byte(packed)
					packed >>= 8
				}
				expected := roadsideHex(t, roadsideGolden)
				expected[13] = byte(39 + 4 + n)
				expected[26] = 1
				expected = append(expected, 0x21, 7, 0, byte(n))
				expected = append(expected, payload...)
				if !bytes.Equal(got, expected) {
					t.Fatalf("value%d: got%x want%x", value, got, expected)
				}
				decoded, e := DecodeRoadsideWithProfile(expected, p)
				if e != nil {
					t.Fatal(e)
				}
				if actual := tc.get(decoded.Groups[0].Objects[0].Extensions[0].Fields); actual != value {
					t.Fatalf("decoded%d want%d", actual, value)
				}
			}
			if tc.width < 32 {
				f := roadsideFixture()
				fields := new(ObjectExtensionFields)
				tc.set(fields, maximum+1)
				f.Groups[0].Objects[0].Extensions = []RoadsideObjectExtension{{ServiceID: 7, Fields: fields}}
				if _, e := NewRoadside(f, p); !errors.Is(e, ErrRange) {
					t.Fatal("overflow", e)
				}
			}
		})
	}
	f := roadsideProfileFixture()
	f.Groups[0].Objects[0].Extensions[0].Fields.Steps = 1
	if _, e := NewRoadside(f, roadsideProfile()); e == nil {
		t.Fatal("absent field silently ignored")
	}
}

func TestRoadsideOpaqueSpanPlacement(t *testing.T) {
	optionFields := func() RoadsideFields {
		f := roadsideFixture()
		f.Groups[0].Objects = nil
		f.Options = []RoadsideOption{{Index: 1, Data: []byte{0xaa}}, {Index: 2, Data: []byte{0xbb}}}
		return f
	}
	for _, tc := range []struct {
		name string
		edit func(*RoadsideMessage)
	}{
		{"option insert", func(m *RoadsideMessage) {
			m.Options = append([]RoadsideOption{{Index: 0, Sensors: []RoadsideSensor{{Attributes: []byte{1}}}}}, m.Options...)
		}},
		{"option remove", func(m *RoadsideMessage) { m.Options = m.Options[1:] }},
		{"option reorder", func(m *RoadsideMessage) {
			m.Options[0], m.Options[1] = m.Options[1], m.Options[0]
			m.Options[0].Index = 1
			m.Options[1].Index = 2
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, e := NewRoadside(optionFields(), RoadsideProfile{})
			if e != nil {
				t.Fatal(e)
			}
			tc.edit(&m)
			if out, e := m.MarshalBinary(); !errors.Is(e, ErrLayout) || out != nil {
				t.Fatalf("relocated opaque span: %x %v", out, e)
			}
		})
	}
	m, e := NewRoadside(optionFields(), RoadsideProfile{})
	if e != nil {
		t.Fatal(e)
	}
	m.Options[0].Data[0] = 0xcc
	m.Options[1].Data = append(m.Options[1].Data, 0xdd)
	if _, e := m.MarshalBinary(); e != nil {
		t.Fatal("same start edits/final growth", e)
	}
	for _, example := range []bool{false, true} {
		t.Run(map[bool]string{false: "raw sensors", true: "example extra"}[example], func(t *testing.T) {
			f := roadsideFixture()
			f.Groups = []RoadsideObjectGroup{{SensorIndex: 0}, {SensorIndex: 1}}
			p := RoadsideProfile{Sensors: []SensorProfile{{}, {}}, GroupOrder: []int{0, 1}}
			f.Options = []RoadsideOption{{Index: 0, Sensors: []RoadsideSensor{{Attributes: []byte{0xaa}}, {Attributes: []byte{0xbb}}}}}
			if example {
				p.Sensors[0].Layout = SensorExample
				p.Sensors[1].Layout = SensorExample
				f.Options[0].Sensors = []RoadsideSensor{{Example: &RoadsideSensorAttributes{ID: 1, Extra: []byte{0xaa}}}, {Example: &RoadsideSensorAttributes{ID: 2, Extra: []byte{0xbb}}}}
			}
			fresh := func() RoadsideMessage {
				t.Helper()
				m, e := NewRoadside(f, p)
				if e != nil {
					t.Fatal(e)
				}
				return m
			}
			m := fresh()
			if example {
				m.Options[0].Sensors[0].Example.Extra = append(m.Options[0].Sensors[0].Example.Extra, 0xcc)
			} else {
				m.Options[0].Sensors[0].Attributes = append(m.Options[0].Sensors[0].Attributes, 0xcc)
			}
			if out, e := m.MarshalBinary(); !errors.Is(e, ErrLayout) || out != nil {
				t.Fatalf("next sensor moved: %x %v", out, e)
			}
			rebuilt, e := NewRoadside(m.RoadsideFields, p)
			if e != nil {
				t.Fatal("explicit sensor reconstruction", e)
			}
			if _, e := rebuilt.MarshalBinary(); e != nil {
				t.Fatal(e)
			}
			if _, e := m.MarshalBinary(); !errors.Is(e, ErrLayout) {
				t.Fatal("reconstruction mutated original anchor", e)
			}
			m = fresh()
			m.Options[0].Sensors[0], m.Options[0].Sensors[1] = m.Options[0].Sensors[1], m.Options[0].Sensors[0]
			if out, e := m.MarshalBinary(); !errors.Is(e, ErrLayout) || out != nil {
				t.Fatalf("reordered sensor: %x %v", out, e)
			}
			m = fresh()
			inserted := RoadsideSensor{Attributes: []byte{0xcc}}
			if example {
				inserted = RoadsideSensor{Example: &RoadsideSensorAttributes{ID: 3}}
			}
			m.Options[0].Sensors = append([]RoadsideSensor{inserted}, m.Options[0].Sensors...)
			if out, e := m.MarshalBinary(); !errors.Is(e, ErrLayout) || out != nil {
				t.Fatalf("inserted sensor moved remainder: %x %v", out, e)
			}
			m = fresh()
			m.Options[0].Sensors = m.Options[0].Sensors[1:]
			if out, e := m.MarshalBinary(); !errors.Is(e, ErrLayout) || out != nil {
				t.Fatalf("removed sensor moved remainder: %x %v", out, e)
			}
			m = fresh()
			if example {
				m.Options[0].Sensors[1].Example.Extra = append(m.Options[0].Sensors[1].Example.Extra, 0xdd)
			} else {
				m.Options[0].Sensors[1].Attributes = append(m.Options[0].Sensors[1].Attributes, 0xdd)
			}
			if _, e := m.MarshalBinary(); e != nil {
				t.Fatal("safe final sensor growth", e)
			}
		})
	}
}
func TestRoadsideKnownSensorMovementAndReconstruction(t *testing.T) {
	f := roadsideFixture()
	f.Options = []RoadsideOption{{Index: 0, Sensors: []RoadsideSensor{{Example: &RoadsideSensorAttributes{ID: 1}}, {Example: &RoadsideSensorAttributes{ID: 2}}}}}
	f.Groups = []RoadsideObjectGroup{{SensorIndex: 0}, {SensorIndex: 1}}
	p := RoadsideProfile{Sensors: []SensorProfile{{Layout: SensorExample}, {Layout: SensorExample}}, GroupOrder: []int{0, 1}}
	m, e := NewRoadside(f, p)
	if e != nil {
		t.Fatal(e)
	}
	m.Options[0].Sensors[0].Example.Extra = []byte{0xaa}
	if _, e := m.MarshalBinary(); e != nil {
		t.Fatal("known second sensor may move", e)
	}
	// Explicit construction clears both outer extension anchors and inherited common anchors.
	f = roadsideFixture()
	f.Groups[0].Objects[0].Extensions = []RoadsideObjectExtension{{ServiceID: 0xee, Data: []byte{1}}}
	m, e = NewRoadside(f, RoadsideProfile{})
	if e != nil {
		t.Fatal(e)
	}
	m.Groups[0].Objects[0].Common.PositionOptional = &rc013.PositionOptional{}
	if _, e := m.MarshalBinary(); !errors.Is(e, ErrLayout) {
		t.Fatal(e)
	}
	if _, e := NewRoadside(m.RoadsideFields, RoadsideProfile{}); e != nil {
		t.Fatal("extension reconstruction", e)
	}
	f = roadsideFixture()
	f.Groups[0].Objects[0].Common.ExtendedOptions = true
	f.Groups[0].Objects[0].Common.OpaqueTail = []byte{0xa5}
	m, e = NewRoadside(f, RoadsideProfile{})
	if e != nil {
		t.Fatal(e)
	}
	m.Groups[0].Objects[0].Common.PositionOptional = &rc013.PositionOptional{}
	if _, e := m.MarshalBinary(); !errors.Is(e, ErrLayout) {
		t.Fatal(e)
	}
	if _, e := NewRoadside(m.RoadsideFields, RoadsideProfile{}); e != nil {
		t.Fatal("common reconstruction", e)
	}
}
