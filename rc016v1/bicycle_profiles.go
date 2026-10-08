package rc016

import "github.com/hareku/its-forum-go/rc016v1/internal/bitio"

// BicycleApplicationKind selects a chapter-3 payload layout, not a wire service ID.
type BicycleApplicationKind uint8

const (
	ApplicationPersonalCommon BicycleApplicationKind = iota + 1
	ApplicationBicycleBasic
	ApplicationBicycleExtended
	ApplicationPedestrian
)

// BicycleProfile assigns experiment-owned free application IDs to fixed layouts.
// Decoders and constructors snapshot this map; later caller changes have no effect.
type BicycleProfile struct {
	Applications map[uint8]BicycleApplicationKind
}

// PersonalCommon is the five-byte level, delay (10 ms), and monitoring frame.
type PersonalCommon struct {
	Level       uint8
	SystemDelay uint8
	WatchData   uint32
}

// MarshalBinary encodes exactly 5 bytes without normalizing raw values.
func (v PersonalCommon) MarshalBinary() ([]byte, error) {
	out := make([]byte, 5)
	if err := bitio.Write(out, 0, 3, uint64(v.Level)); err != nil {
		return nil, &Error{Field: "PersonalCommon.Level", Offset: 0, Err: ErrRange}
	}
	if err := bitio.Write(out, 3, 5, uint64(v.SystemDelay)); err != nil {
		return nil, &Error{Field: "PersonalCommon.SystemDelay", Offset: 0, Err: ErrRange}
	}
	if err := bitio.Write(out, 8, 32, uint64(v.WatchData)); err != nil {
		return nil, &Error{Field: "PersonalCommon.WatchData", Offset: 1, Err: ErrRange}
	}
	return out, nil
}

// UnmarshalBinary decodes an exact frame and leaves the receiver unchanged on error.
func (v *PersonalCommon) UnmarshalBinary(data []byte) error {
	if len(data) != 5 {
		kind := ErrMalformed
		if len(data) < 5 {
			kind = ErrTruncated
		}
		return &Error{Field: "PersonalCommon", Offset: len(data), Err: kind}
	}
	var decoded PersonalCommon
	Level, _ := bitio.Read(data, 0, 3)
	decoded.Level = uint8(Level)
	SystemDelay, _ := bitio.Read(data, 3, 5)
	decoded.SystemDelay = uint8(SystemDelay)
	WatchData, _ := bitio.Read(data, 8, 32)
	decoded.WatchData = uint32(WatchData)
	*v = decoded
	return nil
}

// BicycleBasic is the three-byte basic bicycle frame; DrivePower uses 10 W units.
type BicycleBasic struct {
	AssistType  uint8
	BicycleType uint8
	AssistState uint8
	Pedaling    uint8
	DrivePower  uint8
	Collision   uint8
}

// MarshalBinary encodes exactly 3 bytes without normalizing raw values.
func (v BicycleBasic) MarshalBinary() ([]byte, error) {
	out := make([]byte, 3)
	if err := bitio.Write(out, 0, 4, uint64(v.AssistType)); err != nil {
		return nil, &Error{Field: "BicycleBasic.AssistType", Offset: 0, Err: ErrRange}
	}
	if err := bitio.Write(out, 4, 4, uint64(v.BicycleType)); err != nil {
		return nil, &Error{Field: "BicycleBasic.BicycleType", Offset: 0, Err: ErrRange}
	}
	if err := bitio.Write(out, 8, 2, uint64(v.AssistState)); err != nil {
		return nil, &Error{Field: "BicycleBasic.AssistState", Offset: 1, Err: ErrRange}
	}
	if err := bitio.Write(out, 10, 2, uint64(v.Pedaling)); err != nil {
		return nil, &Error{Field: "BicycleBasic.Pedaling", Offset: 1, Err: ErrRange}
	}
	if err := bitio.Write(out, 12, 8, uint64(v.DrivePower)); err != nil {
		return nil, &Error{Field: "BicycleBasic.DrivePower", Offset: 1, Err: ErrRange}
	}
	if err := bitio.Write(out, 20, 4, uint64(v.Collision)); err != nil {
		return nil, &Error{Field: "BicycleBasic.Collision", Offset: 2, Err: ErrRange}
	}
	return out, nil
}

// UnmarshalBinary decodes an exact frame and leaves the receiver unchanged on error.
func (v *BicycleBasic) UnmarshalBinary(data []byte) error {
	if len(data) != 3 {
		kind := ErrMalformed
		if len(data) < 3 {
			kind = ErrTruncated
		}
		return &Error{Field: "BicycleBasic", Offset: len(data), Err: kind}
	}
	var decoded BicycleBasic
	AssistType, _ := bitio.Read(data, 0, 4)
	decoded.AssistType = uint8(AssistType)
	BicycleType, _ := bitio.Read(data, 4, 4)
	decoded.BicycleType = uint8(BicycleType)
	AssistState, _ := bitio.Read(data, 8, 2)
	decoded.AssistState = uint8(AssistState)
	Pedaling, _ := bitio.Read(data, 10, 2)
	decoded.Pedaling = uint8(Pedaling)
	DrivePower, _ := bitio.Read(data, 12, 8)
	decoded.DrivePower = uint8(DrivePower)
	Collision, _ := bitio.Read(data, 20, 4)
	decoded.Collision = uint8(Collision)
	*v = decoded
	return nil
}

// BicycleExtended is the fourteen-byte extended bicycle frame. Cadence remains a raw integer because the source gives conflicting units. Reserved bits are retained.
type BicycleExtended struct {
	MainGear          uint8
	MainMaxGear       uint8
	SubGear           uint8
	SubMaxGear        uint8
	TireCircumference uint8
	Cadence           uint8
	GearRatio         uint16
	RiderTorque       uint8
	MotorTorque       uint8
	MaxAssistPower    uint8
	AssistPower       uint8
	HumanPower        uint8
	MaxBattery        uint8
	Battery           uint8
	RearLight         uint8
	DriveUnitStatus   uint8
	Maintenance       uint8
	Reserved          uint8
}

// MarshalBinary encodes exactly 14 bytes without normalizing raw values.
func (v BicycleExtended) MarshalBinary() ([]byte, error) {
	out := make([]byte, 14)
	if err := bitio.Write(out, 0, 5, uint64(v.MainGear)); err != nil {
		return nil, &Error{Field: "BicycleExtended.MainGear", Offset: 0, Err: ErrRange}
	}
	if err := bitio.Write(out, 5, 5, uint64(v.MainMaxGear)); err != nil {
		return nil, &Error{Field: "BicycleExtended.MainMaxGear", Offset: 0, Err: ErrRange}
	}
	if err := bitio.Write(out, 10, 5, uint64(v.SubGear)); err != nil {
		return nil, &Error{Field: "BicycleExtended.SubGear", Offset: 1, Err: ErrRange}
	}
	if err := bitio.Write(out, 15, 5, uint64(v.SubMaxGear)); err != nil {
		return nil, &Error{Field: "BicycleExtended.SubMaxGear", Offset: 1, Err: ErrRange}
	}
	if err := bitio.Write(out, 20, 8, uint64(v.TireCircumference)); err != nil {
		return nil, &Error{Field: "BicycleExtended.TireCircumference", Offset: 2, Err: ErrRange}
	}
	if err := bitio.Write(out, 28, 8, uint64(v.Cadence)); err != nil {
		return nil, &Error{Field: "BicycleExtended.Cadence", Offset: 3, Err: ErrRange}
	}
	if err := bitio.Write(out, 36, 10, uint64(v.GearRatio)); err != nil {
		return nil, &Error{Field: "BicycleExtended.GearRatio", Offset: 4, Err: ErrRange}
	}
	if err := bitio.Write(out, 46, 8, uint64(v.RiderTorque)); err != nil {
		return nil, &Error{Field: "BicycleExtended.RiderTorque", Offset: 5, Err: ErrRange}
	}
	if err := bitio.Write(out, 54, 8, uint64(v.MotorTorque)); err != nil {
		return nil, &Error{Field: "BicycleExtended.MotorTorque", Offset: 6, Err: ErrRange}
	}
	if err := bitio.Write(out, 62, 8, uint64(v.MaxAssistPower)); err != nil {
		return nil, &Error{Field: "BicycleExtended.MaxAssistPower", Offset: 7, Err: ErrRange}
	}
	if err := bitio.Write(out, 70, 8, uint64(v.AssistPower)); err != nil {
		return nil, &Error{Field: "BicycleExtended.AssistPower", Offset: 8, Err: ErrRange}
	}
	if err := bitio.Write(out, 78, 8, uint64(v.HumanPower)); err != nil {
		return nil, &Error{Field: "BicycleExtended.HumanPower", Offset: 9, Err: ErrRange}
	}
	if err := bitio.Write(out, 86, 8, uint64(v.MaxBattery)); err != nil {
		return nil, &Error{Field: "BicycleExtended.MaxBattery", Offset: 10, Err: ErrRange}
	}
	if err := bitio.Write(out, 94, 8, uint64(v.Battery)); err != nil {
		return nil, &Error{Field: "BicycleExtended.Battery", Offset: 11, Err: ErrRange}
	}
	if err := bitio.Write(out, 102, 2, uint64(v.RearLight)); err != nil {
		return nil, &Error{Field: "BicycleExtended.RearLight", Offset: 12, Err: ErrRange}
	}
	if err := bitio.Write(out, 104, 2, uint64(v.DriveUnitStatus)); err != nil {
		return nil, &Error{Field: "BicycleExtended.DriveUnitStatus", Offset: 13, Err: ErrRange}
	}
	if err := bitio.Write(out, 106, 2, uint64(v.Maintenance)); err != nil {
		return nil, &Error{Field: "BicycleExtended.Maintenance", Offset: 13, Err: ErrRange}
	}
	if err := bitio.Write(out, 108, 4, uint64(v.Reserved)); err != nil {
		return nil, &Error{Field: "BicycleExtended.Reserved", Offset: 13, Err: ErrRange}
	}
	return out, nil
}

// UnmarshalBinary decodes an exact frame and leaves the receiver unchanged on error.
func (v *BicycleExtended) UnmarshalBinary(data []byte) error {
	if len(data) != 14 {
		kind := ErrMalformed
		if len(data) < 14 {
			kind = ErrTruncated
		}
		return &Error{Field: "BicycleExtended", Offset: len(data), Err: kind}
	}
	var decoded BicycleExtended
	MainGear, _ := bitio.Read(data, 0, 5)
	decoded.MainGear = uint8(MainGear)
	MainMaxGear, _ := bitio.Read(data, 5, 5)
	decoded.MainMaxGear = uint8(MainMaxGear)
	SubGear, _ := bitio.Read(data, 10, 5)
	decoded.SubGear = uint8(SubGear)
	SubMaxGear, _ := bitio.Read(data, 15, 5)
	decoded.SubMaxGear = uint8(SubMaxGear)
	TireCircumference, _ := bitio.Read(data, 20, 8)
	decoded.TireCircumference = uint8(TireCircumference)
	Cadence, _ := bitio.Read(data, 28, 8)
	decoded.Cadence = uint8(Cadence)
	GearRatio, _ := bitio.Read(data, 36, 10)
	decoded.GearRatio = uint16(GearRatio)
	RiderTorque, _ := bitio.Read(data, 46, 8)
	decoded.RiderTorque = uint8(RiderTorque)
	MotorTorque, _ := bitio.Read(data, 54, 8)
	decoded.MotorTorque = uint8(MotorTorque)
	MaxAssistPower, _ := bitio.Read(data, 62, 8)
	decoded.MaxAssistPower = uint8(MaxAssistPower)
	AssistPower, _ := bitio.Read(data, 70, 8)
	decoded.AssistPower = uint8(AssistPower)
	HumanPower, _ := bitio.Read(data, 78, 8)
	decoded.HumanPower = uint8(HumanPower)
	MaxBattery, _ := bitio.Read(data, 86, 8)
	decoded.MaxBattery = uint8(MaxBattery)
	Battery, _ := bitio.Read(data, 94, 8)
	decoded.Battery = uint8(Battery)
	RearLight, _ := bitio.Read(data, 102, 2)
	decoded.RearLight = uint8(RearLight)
	DriveUnitStatus, _ := bitio.Read(data, 104, 2)
	decoded.DriveUnitStatus = uint8(DriveUnitStatus)
	Maintenance, _ := bitio.Read(data, 106, 2)
	decoded.Maintenance = uint8(Maintenance)
	Reserved, _ := bitio.Read(data, 108, 4)
	decoded.Reserved = uint8(Reserved)
	*v = decoded
	return nil
}

// Pedestrian is the five-byte pedestrian frame, including all eighteen reserved bits.
type Pedestrian struct {
	ShoeAttribute uint8
	Steps         uint16
	Motion        uint8
	Reserved      uint32
}

// MarshalBinary encodes exactly 5 bytes without normalizing raw values.
func (v Pedestrian) MarshalBinary() ([]byte, error) {
	out := make([]byte, 5)
	if err := bitio.Write(out, 0, 6, uint64(v.ShoeAttribute)); err != nil {
		return nil, &Error{Field: "Pedestrian.ShoeAttribute", Offset: 0, Err: ErrRange}
	}
	if err := bitio.Write(out, 6, 14, uint64(v.Steps)); err != nil {
		return nil, &Error{Field: "Pedestrian.Steps", Offset: 0, Err: ErrRange}
	}
	if err := bitio.Write(out, 20, 2, uint64(v.Motion)); err != nil {
		return nil, &Error{Field: "Pedestrian.Motion", Offset: 2, Err: ErrRange}
	}
	if err := bitio.Write(out, 22, 18, uint64(v.Reserved)); err != nil {
		return nil, &Error{Field: "Pedestrian.Reserved", Offset: 2, Err: ErrRange}
	}
	return out, nil
}

// UnmarshalBinary decodes an exact frame and leaves the receiver unchanged on error.
func (v *Pedestrian) UnmarshalBinary(data []byte) error {
	if len(data) != 5 {
		kind := ErrMalformed
		if len(data) < 5 {
			kind = ErrTruncated
		}
		return &Error{Field: "Pedestrian", Offset: len(data), Err: kind}
	}
	var decoded Pedestrian
	ShoeAttribute, _ := bitio.Read(data, 0, 6)
	decoded.ShoeAttribute = uint8(ShoeAttribute)
	Steps, _ := bitio.Read(data, 6, 14)
	decoded.Steps = uint16(Steps)
	Motion, _ := bitio.Read(data, 20, 2)
	decoded.Motion = uint8(Motion)
	Reserved, _ := bitio.Read(data, 22, 18)
	decoded.Reserved = uint32(Reserved)
	*v = decoded
	return nil
}
