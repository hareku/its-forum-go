package rc016_test

import (
	"bytes"
	"fmt"
	"github.com/hareku/its-forum-go/rc013v1"
	rc016 "github.com/hareku/its-forum-go/rc016v2"
	"strings"
	"testing"
)

func unavailableCommon() rc013.CommonData {
	return rc013.CommonData{
		Time:              rc013.Time{Hour: 127, Minute: 255, Millisecond: 65535},
		Position:          rc013.Position{Latitude: -2147483648, Longitude: -2147483648, Elevation: 0xf000},
		VehicleState:      rc013.VehicleState{Speed: 65535, Heading: 65535, Acceleration: -32768, Transmission: 7, SteeringWheelAngle: -2048},
		VehicleAttributes: rc013.VehicleAttributes{Width: 1, Length: 1},
	}
}
func TestMessageLevelMatrix(t *testing.T) {
	cases := []struct {
		field string
		max   uint8
		edit  func(*rc013.CommonData)
	}{
		{"time.hour", 4, func(c *rc013.CommonData) { c.Time.Hour = 1 }},
		{"time.minute", 4, func(c *rc013.CommonData) { c.Time.Minute = 1 }},
		{"time.millisecond", 4, func(c *rc013.CommonData) { c.Time.Millisecond = 1 }},
		{"time.leapSecondCorrection", 4, func(c *rc013.CommonData) { c.Time.LeapSecondCorrection = true }},
		{"position.latitude", 3, func(c *rc013.CommonData) { c.Position.Latitude = 1 }},
		{"position.longitude", 3, func(c *rc013.CommonData) { c.Position.Longitude = 1 }},
		{"position.elevation", 3, func(c *rc013.CommonData) { c.Position.Elevation = 1 }},
		{"position.positionConfidence", 3, func(c *rc013.CommonData) { c.Position.PositionConfidence = 15 }},
		{"position.elevationConfidence", 3, func(c *rc013.CommonData) { c.Position.ElevationConfidence = 15 }},
		{"vehicleState.speed", 1, func(c *rc013.CommonData) { c.VehicleState.Speed = 1 }},
		{"vehicleState.acceleration", 1, func(c *rc013.CommonData) { c.VehicleState.Acceleration = 1 }},
		{"vehicleState.speedConfidence", 1, func(c *rc013.CommonData) { c.VehicleState.SpeedConfidence = 7 }},
		{"vehicleState.accelerationConfidence", 1, func(c *rc013.CommonData) { c.VehicleState.AccelerationConfidence = 7 }},
		{"vehicleState.heading", 2, func(c *rc013.CommonData) { c.VehicleState.Heading = 1 }},
		{"vehicleState.headingConfidence", 2, func(c *rc013.CommonData) { c.VehicleState.HeadingConfidence = 7 }},
		{"vehicleState.transmission", 2, func(c *rc013.CommonData) { c.VehicleState.Transmission = 0 }},
		{"vehicleState.steeringWheelAngle", 2, func(c *rc013.CommonData) { c.VehicleState.SteeringWheelAngle = 1 }},
	}
	for level := uint8(0); level <= 7; level++ {
		for _, ped := range []bool{false, true} {
			f := messageFields(ped)
			f.Common = unavailableCommon()
			if ped {
				f.Applications[0].Pedestrian.PersonalCommon.Level = level
				f.Applications[0].Pedestrian.Pedestrian.Reserved = 0
			} else {
				f.Applications[0].Bicycle.PersonalCommon.Level = level
				f.Applications[0].Bicycle.Extended.Reserved = 0
			}
			m, e := rc016.NewMessage(f, messageProfile())
			if e != nil {
				t.Fatal(e)
			}
			base := m.Validate()
			want := 0
			if level == 0 || level == 6 {
				want = 1
			}
			if len(base) != want {
				t.Fatalf("level %d sentinel issues %v", level, base)
			}
			for _, tc := range cases {
				t.Run(fmt.Sprintf("%d/%v/%s", level, ped, tc.field), func(t *testing.T) {
					m.Common = unavailableCommon()
					tc.edit(&m.Common)
					before := messageBytes(t, m)
					issues := m.Validate()
					found := false
					for _, issue := range issues {
						if issue.Field == "message.applications[0].common."+tc.field {
							found = true
						}
					}
					if found != (level >= 1 && level <= tc.max) {
						t.Fatalf("level rule %v", issues)
					}
					if !bytes.Equal(before, messageBytes(t, m)) {
						t.Fatal("validation mutated message")
					}
				})
			}
		}
	}
}
func TestMessageMultipleLevelsAndBaseValidation(t *testing.T) {
	f := messageFields(false)
	f.Applications[0].Bicycle.PersonalCommon.Level = 1
	f.Applications = append(f.Applications, messageFields(true).Applications[0])
	m, e := rc016.NewMessage(f, messageProfile())
	if e != nil {
		t.Fatal(e)
	}
	var first, second bool
	for _, issue := range m.Validate() {
		if strings.HasPrefix(issue.Field, "message.applications[0].common.") {
			first = true
		}
		if strings.HasPrefix(issue.Field, "message.applications[1].common.") {
			second = true
		}
	}
	if !first || second {
		t.Fatal("levels not independent")
	}
	m.Common.Time.Hour = 100
	found := false
	for _, issue := range m.Validate() {
		if issue.Field == "time.hour" {
			found = true
		}
	}
	if !found {
		t.Fatal("RC013 diagnostics omitted")
	}
}
