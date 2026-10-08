// Package rc013 encodes and decodes ITS FORUM RC-013 version 1.1 messages.
// Values are encoded integers: decoding does not normalize unknown values,
// reserved bits, or unavailable sentinels. The codec uses MSB-first big-endian
// fields, not ASN.1 encoding. See ITS FORUM RC-013 1.1, chapters 4 through 6.
package rc013

// Latitude is WGS84 latitude in 0.0000001 degree units.
type Latitude int32

// Longitude is WGS84 longitude in 0.0000001 degree units.
type Longitude int32

// Elevation preserves the special unsigned 16-bit representation from section 6.3.3.
type Elevation uint16

// Speed is measured in 0.01 m/s units.
type Speed uint16

// Heading is measured clockwise from north in 0.0125 degree units.
type Heading uint16

// Acceleration is measured in 0.01 m/s squared units.
type Acceleration int16

// SteeringWheelAngle is measured clockwise in 1.5 degree units and fits signed 12 bits.
type SteeringWheelAngle int16

// YawRate is measured clockwise in 0.01 degree/s units.
type YawRate int16

const (
	LatitudeUnavailable           Latitude           = -2147483648
	LongitudeUnavailable          Longitude          = -2147483648
	ElevationUnavailable          Elevation          = 0xf000
	SpeedUnavailable              Speed              = 0xffff
	HeadingUnavailable            Heading            = 0xffff
	AccelerationUnavailable       Acceleration       = -32768
	SteeringWheelAngleUnavailable SteeringWheelAngle = -2048
	YawRateUnavailable            YawRate            = -32768
	HourUnavailable               uint8              = 127
	MinuteUnavailable             uint8              = 255
	MillisecondUnavailable        uint16             = 65535
)

// Decimeters returns the physical elevation, or false for the unavailable sentinel.
// Values 0x8000..0xefff are positive, unlike ordinary signed 16-bit integers.
func (v Elevation) Decimeters() (int32, bool) {
	if v == ElevationUnavailable {
		return 0, false
	}
	if v > ElevationUnavailable {
		return int32(v) - 65536, true
	}
	return int32(v), true
}

// ElevationFromDecimeters encodes an exact value without rounding or saturation.
func ElevationFromDecimeters(v int32) (Elevation, error) {
	if v < -4095 || v > 61439 {
		return 0, fieldError("elevation", -1, ErrRange)
	}
	return Elevation(uint16(v)), nil
}
