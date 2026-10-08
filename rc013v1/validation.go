package rc013

// Validate reports meaningful-range and reserved-value concerns. Unavailable
// sentinels are valid representations, not validation errors. The acceleration
// specification has conflicting physical-range prose and ASN.1 annotations;
// its signed wire value is preserved without imposing either conflicting range.
func (c CommonData) Validate() []Issue {
	var issues []Issue
	add := func(field, message string, bad bool) {
		if bad {
			issues = append(issues, Issue{Field: field, Message: message})
		}
	}
	add("time.hour", "hour is outside 0..23", c.Time.Hour > 23 && c.Time.Hour != HourUnavailable)
	add("time.minute", "minute is outside 0..59", c.Time.Minute > 59 && c.Time.Minute != MinuteUnavailable)
	add("time.millisecond", "millisecond is outside 0..60999", c.Time.Millisecond > 60999 && c.Time.Millisecond != MillisecondUnavailable)
	latitude := func(field string, v Latitude) {
		add(field, "latitude is outside -90..90 degrees", v != LatitudeUnavailable && (v < -900000000 || v > 900000000))
	}
	longitude := func(field string, v Longitude) {
		add(field, "longitude is outside -180..180 degrees", v != LongitudeUnavailable && (v < -1800000000 || v > 1800000000))
	}
	heading := func(field string, v Heading) {
		add(field, "heading is outside 0..359.9875 degrees", v != HeadingUnavailable && v > 28799)
	}
	latitude("position.latitude", c.Position.Latitude)
	longitude("position.longitude", c.Position.Longitude)
	add("vehicleState.speed", "speed is outside 0..163.83 m/s", c.VehicleState.Speed > 16383 && c.VehicleState.Speed != SpeedUnavailable)
	heading("vehicleState.heading", c.VehicleState.Heading)
	add("vehicleState.transmission", "reserved transmission value", c.VehicleState.Transmission >= 4 && c.VehicleState.Transmission <= 6)
	add("vehicleAttributes.sizeClass", "reserved size class", c.VehicleAttributes.SizeClass >= 8 && c.VehicleAttributes.SizeClass <= 14)
	add("vehicleAttributes.roleClass", "reserved role class", c.VehicleAttributes.RoleClass >= 6 && c.VehicleAttributes.RoleClass <= 14)
	add("vehicleAttributes.width", "zero width is unassigned", c.VehicleAttributes.Width == 0)
	add("vehicleAttributes.length", "zero length is unassigned", c.VehicleAttributes.Length == 0)
	if p := c.PositionOptional; p != nil {
		add("positionOptional.delay", "zero delay is unassigned", p.Delay == 0)
		add("positionOptional.revision", "zero revision is unassigned", p.Revision == 0)
		add("positionOptional.roadFacilities", "reserved road facility", p.RoadFacilities == 5 || p.RoadFacilities == 6)
		add("positionOptional.roadClass", "reserved road class", p.RoadClass == 7)
	}
	if p := c.GPSStatusOptional; p != nil {
		heading("gpsStatusOptional.orientation", p.Orientation)
	}
	if p := c.PositionAcquisitionOptional; p != nil {
		add("positionAcquisitionOptional.multipath", "reserved multipath value", p.Multipath == 3)
	}
	if p := c.VehicleStateOptional; p != nil {
		add("vehicleStateOptional.auxiliaryBrakes", "reserved auxiliary brake value", p.AuxiliaryBrakes == 3)
		add("vehicleStateOptional.throttle", "throttle is outside 0..100 percent", p.Throttle > 200 && p.Throttle != 255)
		add("vehicleStateOptional.lights", "reserved light bit is set", p.Lights&1 != 0)
	}
	if p := c.Intersection; p != nil {
		add("intersection.distanceSource", "reserved distance source", p.DistanceSource > 2)
		add("intersection.positionSource", "reserved position source", p.PositionSource > 2)
		add("intersection.distance", "distance is outside 0..1000 meters", p.Distance > 1000 && p.Distance != 1023)
		latitude("intersection.latitude", p.Latitude)
		longitude("intersection.longitude", p.Longitude)
	}
	if p := c.Extension; p != nil {
		var maxUpper, maxStatus uint8
		known := true
		switch c.VehicleAttributes.RoleClass {
		case 0:
			maxUpper, maxStatus = 7, 4
		case 1:
			maxUpper, maxStatus = 0, 2
		case 2:
			maxUpper, maxStatus = 2, 5
		case 3:
			maxUpper, maxStatus = 4, 5
		case 4, 5:
			maxUpper, maxStatus = 0, 1
		case 15:
			maxUpper, maxStatus = 0, 0
		default:
			known = false
		}
		if known {
			add("extension.upper", "reserved upper extension value", p.Upper > maxUpper)
			add("extension.status", "reserved extension status", p.Status > maxStatus && p.Status != 15)
		}
	}
	return issues
}
