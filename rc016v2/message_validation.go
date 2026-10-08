package rc016

import (
	"fmt"
	"github.com/hareku/its-forum-go/rc013v1"
)

// Validate reports semantic concerns without changing wire values. The equipment
// level rules from RC-016 table 3-1 are diagnostic policy, not codec rejection.
func (m Message) Validate() []Issue {
	issues := (rc013.Message{Header: m.Header, Common: m.Common}).Validate()
	for i, a := range m.Applications {
		var payloadIssues []Issue
		var level uint8
		var kind string
		switch m.profile.Applications[a.ServiceID] {
		case ApplicationBicycle:
			if a.Bicycle == nil {
				continue
			}
			payloadIssues, level, kind = a.Bicycle.Validate(), a.Bicycle.PersonalCommon.Level, "bicycle"
		case ApplicationPedestrian:
			if a.Pedestrian == nil {
				continue
			}
			payloadIssues, level, kind = a.Pedestrian.Validate(), a.Pedestrian.PersonalCommon.Level, "pedestrian"
		default:
			continue
		}
		prefix := fmt.Sprintf("message.applications[%d].", i)
		for _, issue := range payloadIssues {
			issue.Field = prefix + kind + "." + issue.Field
			issues = append(issues, issue)
		}
		for _, issue := range levelIssues(m.Common, level) {
			issue.Field = prefix + "common." + issue.Field
			issues = append(issues, issue)
		}
	}
	return issues
}

func levelIssues(c rc013.CommonData, level uint8) []Issue {
	if level < 1 || level > 5 {
		return nil
	}
	var issues []Issue
	add := func(field string, bad bool) {
		if bad {
			issues = append(issues, Issue{Field: field, Message: "equipment level requires the RC-013 unavailable value"})
		}
	}
	if level <= 4 {
		add("time.hour", c.Time.Hour != rc013.HourUnavailable)
		add("time.minute", c.Time.Minute != rc013.MinuteUnavailable)
		add("time.millisecond", c.Time.Millisecond != rc013.MillisecondUnavailable)
		if c.Time.LeapSecondCorrection {
			issues = append(issues, Issue{Field: "time.leapSecondCorrection", Message: "equipment level requires zero leap-second correction"})
		}
	}
	// Confidence unavailable values are zero, not all bits set: RC-013
	// sections 6.3.4, 6.3.5, 6.4.4, 6.4.5, and 6.4.6.
	if level <= 3 {
		add("position.latitude", c.Position.Latitude != rc013.LatitudeUnavailable)
		add("position.longitude", c.Position.Longitude != rc013.LongitudeUnavailable)
		add("position.elevation", c.Position.Elevation != rc013.ElevationUnavailable)
		add("position.positionConfidence", c.Position.PositionConfidence != 0)
		add("position.elevationConfidence", c.Position.ElevationConfidence != 0)
	}
	if level == 1 {
		add("vehicleState.speed", c.VehicleState.Speed != rc013.SpeedUnavailable)
		add("vehicleState.acceleration", c.VehicleState.Acceleration != rc013.AccelerationUnavailable)
		add("vehicleState.speedConfidence", c.VehicleState.SpeedConfidence != 0)
		add("vehicleState.accelerationConfidence", c.VehicleState.AccelerationConfidence != 0)
	}
	if level <= 2 {
		add("vehicleState.heading", c.VehicleState.Heading != rc013.HeadingUnavailable)
		add("vehicleState.headingConfidence", c.VehicleState.HeadingConfidence != 0)
		add("vehicleState.transmission", c.VehicleState.Transmission != 7)
		add("vehicleState.steeringWheelAngle", c.VehicleState.SteeringWheelAngle != rc013.SteeringWheelAngleUnavailable)
	}
	return issues
}
