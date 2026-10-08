package rc013

// Header is the 8-byte RC-013 Header data frame.
type Header struct {
	ServiceID   uint8
	MessageID   uint8
	Version     uint8
	VehicleID   uint32
	Counter     uint8
	DataLength  uint8
	OptionFlags uint8
}

// MarshalBinary encodes every field, preserving representable reserved values.
func (v Header) MarshalBinary() ([]byte, error) {
	c := frameCodec{data: make([]byte, 8)}
	c.write("Header.ServiceID", 3, uint64(v.ServiceID))
	c.write("Header.MessageID", 2, uint64(v.MessageID))
	c.write("Header.Version", 3, uint64(v.Version))
	c.write("Header.VehicleID", 32, uint64(v.VehicleID))
	c.write("Header.Counter", 8, uint64(v.Counter))
	c.write("Header.DataLength", 8, uint64(v.DataLength))
	c.write("Header.OptionFlags", 8, uint64(v.OptionFlags))
	if c.err != nil {
		return nil, c.err
	}
	return c.data, nil
}

// UnmarshalBinary requires exactly 8 bytes and leaves the receiver unchanged on error.
func (v *Header) UnmarshalBinary(data []byte) error {
	if v == nil {
		return fieldError("Header", -1, ErrMalformed)
	}
	if err := exactFrame(data, 8, "Header"); err != nil {
		return err
	}
	c := frameCodec{data: data}
	var next Header
	next.ServiceID = uint8(c.read("Header.ServiceID", 3))
	next.MessageID = uint8(c.read("Header.MessageID", 2))
	next.Version = uint8(c.read("Header.Version", 3))
	next.VehicleID = uint32(c.read("Header.VehicleID", 32))
	next.Counter = uint8(c.read("Header.Counter", 8))
	next.DataLength = uint8(c.read("Header.DataLength", 8))
	next.OptionFlags = uint8(c.read("Header.OptionFlags", 8))
	if c.err != nil {
		return c.err
	}
	*v = next
	return nil
}

// Time is the 4-byte RC-013 Time data frame.
type Time struct {
	LeapSecondCorrection bool
	Hour                 uint8
	Minute               uint8
	Millisecond          uint16
}

// MarshalBinary encodes every field, preserving representable reserved values.
func (v Time) MarshalBinary() ([]byte, error) {
	c := frameCodec{data: make([]byte, 4)}
	c.write("Time.LeapSecondCorrection", 1, boolValue(v.LeapSecondCorrection))
	c.write("Time.Hour", 7, uint64(v.Hour))
	c.write("Time.Minute", 8, uint64(v.Minute))
	c.write("Time.Millisecond", 16, uint64(v.Millisecond))
	if c.err != nil {
		return nil, c.err
	}
	return c.data, nil
}

// UnmarshalBinary requires exactly 4 bytes and leaves the receiver unchanged on error.
func (v *Time) UnmarshalBinary(data []byte) error {
	if v == nil {
		return fieldError("Time", -1, ErrMalformed)
	}
	if err := exactFrame(data, 4, "Time"); err != nil {
		return err
	}
	c := frameCodec{data: data}
	var next Time
	next.LeapSecondCorrection = c.read("Time.LeapSecondCorrection", 1) != 0
	next.Hour = uint8(c.read("Time.Hour", 7))
	next.Minute = uint8(c.read("Time.Minute", 8))
	next.Millisecond = uint16(c.read("Time.Millisecond", 16))
	if c.err != nil {
		return c.err
	}
	*v = next
	return nil
}

// Position is the 11-byte RC-013 Position data frame.
type Position struct {
	Latitude            Latitude
	Longitude           Longitude
	Elevation           Elevation
	PositionConfidence  uint8
	ElevationConfidence uint8
}

// MarshalBinary encodes every field, preserving representable reserved values.
func (v Position) MarshalBinary() ([]byte, error) {
	c := frameCodec{data: make([]byte, 11)}
	c.writeSigned("Position.Latitude", 32, int64(v.Latitude))
	c.writeSigned("Position.Longitude", 32, int64(v.Longitude))
	c.write("Position.Elevation", 16, uint64(v.Elevation))
	c.write("Position.PositionConfidence", 4, uint64(v.PositionConfidence))
	c.write("Position.ElevationConfidence", 4, uint64(v.ElevationConfidence))
	if c.err != nil {
		return nil, c.err
	}
	return c.data, nil
}

// UnmarshalBinary requires exactly 11 bytes and leaves the receiver unchanged on error.
func (v *Position) UnmarshalBinary(data []byte) error {
	if v == nil {
		return fieldError("Position", -1, ErrMalformed)
	}
	if err := exactFrame(data, 11, "Position"); err != nil {
		return err
	}
	c := frameCodec{data: data}
	var next Position
	next.Latitude = Latitude(c.signed("Position.Latitude", 32))
	next.Longitude = Longitude(c.signed("Position.Longitude", 32))
	next.Elevation = Elevation(c.read("Position.Elevation", 16))
	next.PositionConfidence = uint8(c.read("Position.PositionConfidence", 4))
	next.ElevationConfidence = uint8(c.read("Position.ElevationConfidence", 4))
	if c.err != nil {
		return c.err
	}
	*v = next
	return nil
}

// VehicleState is the 9-byte RC-013 VehicleState data frame.
type VehicleState struct {
	Speed                  Speed
	Heading                Heading
	Acceleration           Acceleration
	SpeedConfidence        uint8
	HeadingConfidence      uint8
	AccelerationConfidence uint8
	Transmission           uint8
	SteeringWheelAngle     SteeringWheelAngle
}

// MarshalBinary encodes every field, preserving representable reserved values.
func (v VehicleState) MarshalBinary() ([]byte, error) {
	c := frameCodec{data: make([]byte, 9)}
	c.write("VehicleState.Speed", 16, uint64(v.Speed))
	c.write("VehicleState.Heading", 16, uint64(v.Heading))
	c.writeSigned("VehicleState.Acceleration", 16, int64(v.Acceleration))
	c.write("VehicleState.SpeedConfidence", 3, uint64(v.SpeedConfidence))
	c.write("VehicleState.HeadingConfidence", 3, uint64(v.HeadingConfidence))
	c.write("VehicleState.AccelerationConfidence", 3, uint64(v.AccelerationConfidence))
	c.write("VehicleState.Transmission", 3, uint64(v.Transmission))
	c.writeSigned("VehicleState.SteeringWheelAngle", 12, int64(v.SteeringWheelAngle))
	if c.err != nil {
		return nil, c.err
	}
	return c.data, nil
}

// UnmarshalBinary requires exactly 9 bytes and leaves the receiver unchanged on error.
func (v *VehicleState) UnmarshalBinary(data []byte) error {
	if v == nil {
		return fieldError("VehicleState", -1, ErrMalformed)
	}
	if err := exactFrame(data, 9, "VehicleState"); err != nil {
		return err
	}
	c := frameCodec{data: data}
	var next VehicleState
	next.Speed = Speed(c.read("VehicleState.Speed", 16))
	next.Heading = Heading(c.read("VehicleState.Heading", 16))
	next.Acceleration = Acceleration(c.signed("VehicleState.Acceleration", 16))
	next.SpeedConfidence = uint8(c.read("VehicleState.SpeedConfidence", 3))
	next.HeadingConfidence = uint8(c.read("VehicleState.HeadingConfidence", 3))
	next.AccelerationConfidence = uint8(c.read("VehicleState.AccelerationConfidence", 3))
	next.Transmission = uint8(c.read("VehicleState.Transmission", 3))
	next.SteeringWheelAngle = SteeringWheelAngle(c.signed("VehicleState.SteeringWheelAngle", 12))
	if c.err != nil {
		return c.err
	}
	*v = next
	return nil
}

// VehicleAttributes is the 4-byte RC-013 VehicleAttributes data frame.
type VehicleAttributes struct {
	SizeClass uint8
	RoleClass uint8
	Width     uint16
	Length    uint16
}

// MarshalBinary encodes every field, preserving representable reserved values.
func (v VehicleAttributes) MarshalBinary() ([]byte, error) {
	c := frameCodec{data: make([]byte, 4)}
	c.write("VehicleAttributes.SizeClass", 4, uint64(v.SizeClass))
	c.write("VehicleAttributes.RoleClass", 4, uint64(v.RoleClass))
	c.write("VehicleAttributes.Width", 10, uint64(v.Width))
	c.write("VehicleAttributes.Length", 14, uint64(v.Length))
	if c.err != nil {
		return nil, c.err
	}
	return c.data, nil
}

// UnmarshalBinary requires exactly 4 bytes and leaves the receiver unchanged on error.
func (v *VehicleAttributes) UnmarshalBinary(data []byte) error {
	if v == nil {
		return fieldError("VehicleAttributes", -1, ErrMalformed)
	}
	if err := exactFrame(data, 4, "VehicleAttributes"); err != nil {
		return err
	}
	c := frameCodec{data: data}
	var next VehicleAttributes
	next.SizeClass = uint8(c.read("VehicleAttributes.SizeClass", 4))
	next.RoleClass = uint8(c.read("VehicleAttributes.RoleClass", 4))
	next.Width = uint16(c.read("VehicleAttributes.Width", 10))
	next.Length = uint16(c.read("VehicleAttributes.Length", 14))
	if c.err != nil {
		return c.err
	}
	*v = next
	return nil
}

// PositionOptional is the 2-byte RC-013 PositionOptional data frame.
type PositionOptional struct {
	Delay          uint8
	Revision       uint8
	RoadFacilities uint8
	RoadClass      uint8
}

// MarshalBinary encodes every field, preserving representable reserved values.
func (v PositionOptional) MarshalBinary() ([]byte, error) {
	c := frameCodec{data: make([]byte, 2)}
	c.write("PositionOptional.Delay", 5, uint64(v.Delay))
	c.write("PositionOptional.Revision", 5, uint64(v.Revision))
	c.write("PositionOptional.RoadFacilities", 3, uint64(v.RoadFacilities))
	c.write("PositionOptional.RoadClass", 3, uint64(v.RoadClass))
	if c.err != nil {
		return nil, c.err
	}
	return c.data, nil
}

// UnmarshalBinary requires exactly 2 bytes and leaves the receiver unchanged on error.
func (v *PositionOptional) UnmarshalBinary(data []byte) error {
	if v == nil {
		return fieldError("PositionOptional", -1, ErrMalformed)
	}
	if err := exactFrame(data, 2, "PositionOptional"); err != nil {
		return err
	}
	c := frameCodec{data: data}
	var next PositionOptional
	next.Delay = uint8(c.read("PositionOptional.Delay", 5))
	next.Revision = uint8(c.read("PositionOptional.Revision", 5))
	next.RoadFacilities = uint8(c.read("PositionOptional.RoadFacilities", 3))
	next.RoadClass = uint8(c.read("PositionOptional.RoadClass", 3))
	if c.err != nil {
		return c.err
	}
	*v = next
	return nil
}

// GPSStatusOptional is the 4-byte RC-013 GPSStatusOptional data frame.
type GPSStatusOptional struct {
	SemiMajorAxis uint8
	SemiMinorAxis uint8
	Orientation   Heading
}

// MarshalBinary encodes every field, preserving representable reserved values.
func (v GPSStatusOptional) MarshalBinary() ([]byte, error) {
	c := frameCodec{data: make([]byte, 4)}
	c.write("GPSStatusOptional.SemiMajorAxis", 8, uint64(v.SemiMajorAxis))
	c.write("GPSStatusOptional.SemiMinorAxis", 8, uint64(v.SemiMinorAxis))
	c.write("GPSStatusOptional.Orientation", 16, uint64(v.Orientation))
	if c.err != nil {
		return nil, c.err
	}
	return c.data, nil
}

// UnmarshalBinary requires exactly 4 bytes and leaves the receiver unchanged on error.
func (v *GPSStatusOptional) UnmarshalBinary(data []byte) error {
	if v == nil {
		return fieldError("GPSStatusOptional", -1, ErrMalformed)
	}
	if err := exactFrame(data, 4, "GPSStatusOptional"); err != nil {
		return err
	}
	c := frameCodec{data: data}
	var next GPSStatusOptional
	next.SemiMajorAxis = uint8(c.read("GPSStatusOptional.SemiMajorAxis", 8))
	next.SemiMinorAxis = uint8(c.read("GPSStatusOptional.SemiMinorAxis", 8))
	next.Orientation = Heading(c.read("GPSStatusOptional.Orientation", 16))
	if c.err != nil {
		return c.err
	}
	*v = next
	return nil
}

// PositionAcquisitionOptional is the 2-byte RC-013 PositionAcquisitionOptional data frame.
type PositionAcquisitionOptional struct {
	Mode          uint8
	PDOP          uint8
	Satellites    uint8
	Multipath     uint8
	DeadReckoning bool
	MapMatching   bool
}

// MarshalBinary encodes every field, preserving representable reserved values.
func (v PositionAcquisitionOptional) MarshalBinary() ([]byte, error) {
	c := frameCodec{data: make([]byte, 2)}
	c.write("PositionAcquisitionOptional.Mode", 2, uint64(v.Mode))
	c.write("PositionAcquisitionOptional.PDOP", 6, uint64(v.PDOP))
	c.write("PositionAcquisitionOptional.Satellites", 4, uint64(v.Satellites))
	c.write("PositionAcquisitionOptional.Multipath", 2, uint64(v.Multipath))
	c.write("PositionAcquisitionOptional.DeadReckoning", 1, boolValue(v.DeadReckoning))
	c.write("PositionAcquisitionOptional.MapMatching", 1, boolValue(v.MapMatching))
	if c.err != nil {
		return nil, c.err
	}
	return c.data, nil
}

// UnmarshalBinary requires exactly 2 bytes and leaves the receiver unchanged on error.
func (v *PositionAcquisitionOptional) UnmarshalBinary(data []byte) error {
	if v == nil {
		return fieldError("PositionAcquisitionOptional", -1, ErrMalformed)
	}
	if err := exactFrame(data, 2, "PositionAcquisitionOptional"); err != nil {
		return err
	}
	c := frameCodec{data: data}
	var next PositionAcquisitionOptional
	next.Mode = uint8(c.read("PositionAcquisitionOptional.Mode", 2))
	next.PDOP = uint8(c.read("PositionAcquisitionOptional.PDOP", 6))
	next.Satellites = uint8(c.read("PositionAcquisitionOptional.Satellites", 4))
	next.Multipath = uint8(c.read("PositionAcquisitionOptional.Multipath", 2))
	next.DeadReckoning = c.read("PositionAcquisitionOptional.DeadReckoning", 1) != 0
	next.MapMatching = c.read("PositionAcquisitionOptional.MapMatching", 1) != 0
	if c.err != nil {
		return c.err
	}
	*v = next
	return nil
}

// VehicleStateOptional is the 7-byte RC-013 VehicleStateOptional data frame.
type VehicleStateOptional struct {
	YawRate         YawRate
	Brakes          uint8
	AuxiliaryBrakes uint8
	Throttle        uint8
	Lights          uint8
	ACC             uint8
	CACC            uint8
	PCS             uint8
	ABS             uint8
	TRC             uint8
	ESC             uint8
	LKA             uint8
	LDW             uint8
}

// MarshalBinary encodes every field, preserving representable reserved values.
func (v VehicleStateOptional) MarshalBinary() ([]byte, error) {
	c := frameCodec{data: make([]byte, 7)}
	c.writeSigned("VehicleStateOptional.YawRate", 16, int64(v.YawRate))
	c.write("VehicleStateOptional.Brakes", 6, uint64(v.Brakes))
	c.write("VehicleStateOptional.AuxiliaryBrakes", 2, uint64(v.AuxiliaryBrakes))
	c.write("VehicleStateOptional.Throttle", 8, uint64(v.Throttle))
	c.write("VehicleStateOptional.Lights", 8, uint64(v.Lights))
	c.write("VehicleStateOptional.ACC", 2, uint64(v.ACC))
	c.write("VehicleStateOptional.CACC", 2, uint64(v.CACC))
	c.write("VehicleStateOptional.PCS", 2, uint64(v.PCS))
	c.write("VehicleStateOptional.ABS", 2, uint64(v.ABS))
	c.write("VehicleStateOptional.TRC", 2, uint64(v.TRC))
	c.write("VehicleStateOptional.ESC", 2, uint64(v.ESC))
	c.write("VehicleStateOptional.LKA", 2, uint64(v.LKA))
	c.write("VehicleStateOptional.LDW", 2, uint64(v.LDW))
	if c.err != nil {
		return nil, c.err
	}
	return c.data, nil
}

// UnmarshalBinary requires exactly 7 bytes and leaves the receiver unchanged on error.
func (v *VehicleStateOptional) UnmarshalBinary(data []byte) error {
	if v == nil {
		return fieldError("VehicleStateOptional", -1, ErrMalformed)
	}
	if err := exactFrame(data, 7, "VehicleStateOptional"); err != nil {
		return err
	}
	c := frameCodec{data: data}
	var next VehicleStateOptional
	next.YawRate = YawRate(c.signed("VehicleStateOptional.YawRate", 16))
	next.Brakes = uint8(c.read("VehicleStateOptional.Brakes", 6))
	next.AuxiliaryBrakes = uint8(c.read("VehicleStateOptional.AuxiliaryBrakes", 2))
	next.Throttle = uint8(c.read("VehicleStateOptional.Throttle", 8))
	next.Lights = uint8(c.read("VehicleStateOptional.Lights", 8))
	next.ACC = uint8(c.read("VehicleStateOptional.ACC", 2))
	next.CACC = uint8(c.read("VehicleStateOptional.CACC", 2))
	next.PCS = uint8(c.read("VehicleStateOptional.PCS", 2))
	next.ABS = uint8(c.read("VehicleStateOptional.ABS", 2))
	next.TRC = uint8(c.read("VehicleStateOptional.TRC", 2))
	next.ESC = uint8(c.read("VehicleStateOptional.ESC", 2))
	next.LKA = uint8(c.read("VehicleStateOptional.LKA", 2))
	next.LDW = uint8(c.read("VehicleStateOptional.LDW", 2))
	if c.err != nil {
		return c.err
	}
	*v = next
	return nil
}

// Intersection is the 10-byte RC-013 Intersection data frame.
type Intersection struct {
	DistanceSource uint8
	Distance       uint16
	PositionSource uint8
	Latitude       Latitude
	Longitude      Longitude
}

// MarshalBinary encodes every field, preserving representable reserved values.
func (v Intersection) MarshalBinary() ([]byte, error) {
	c := frameCodec{data: make([]byte, 10)}
	c.write("Intersection.DistanceSource", 3, uint64(v.DistanceSource))
	c.write("Intersection.Distance", 10, uint64(v.Distance))
	c.write("Intersection.PositionSource", 3, uint64(v.PositionSource))
	c.writeSigned("Intersection.Latitude", 32, int64(v.Latitude))
	c.writeSigned("Intersection.Longitude", 32, int64(v.Longitude))
	if c.err != nil {
		return nil, c.err
	}
	return c.data, nil
}

// UnmarshalBinary requires exactly 10 bytes and leaves the receiver unchanged on error.
func (v *Intersection) UnmarshalBinary(data []byte) error {
	if v == nil {
		return fieldError("Intersection", -1, ErrMalformed)
	}
	if err := exactFrame(data, 10, "Intersection"); err != nil {
		return err
	}
	c := frameCodec{data: data}
	var next Intersection
	next.DistanceSource = uint8(c.read("Intersection.DistanceSource", 3))
	next.Distance = uint16(c.read("Intersection.Distance", 10))
	next.PositionSource = uint8(c.read("Intersection.PositionSource", 3))
	next.Latitude = Latitude(c.signed("Intersection.Latitude", 32))
	next.Longitude = Longitude(c.signed("Intersection.Longitude", 32))
	if c.err != nil {
		return c.err
	}
	*v = next
	return nil
}

// Extension is the 1-byte RC-013 Extension data frame.
type Extension struct {
	Upper  uint8
	Status uint8
}

// MarshalBinary encodes every field, preserving representable reserved values.
func (v Extension) MarshalBinary() ([]byte, error) {
	c := frameCodec{data: make([]byte, 1)}
	c.write("Extension.Upper", 4, uint64(v.Upper))
	c.write("Extension.Status", 4, uint64(v.Status))
	if c.err != nil {
		return nil, c.err
	}
	return c.data, nil
}

// UnmarshalBinary requires exactly 1 bytes and leaves the receiver unchanged on error.
func (v *Extension) UnmarshalBinary(data []byte) error {
	if v == nil {
		return fieldError("Extension", -1, ErrMalformed)
	}
	if err := exactFrame(data, 1, "Extension"); err != nil {
		return err
	}
	c := frameCodec{data: data}
	var next Extension
	next.Upper = uint8(c.read("Extension.Upper", 4))
	next.Status = uint8(c.read("Extension.Status", 4))
	if c.err != nil {
		return c.err
	}
	*v = next
	return nil
}
