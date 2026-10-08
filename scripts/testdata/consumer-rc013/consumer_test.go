package consumer_test

import (
	"testing"

	rc013 "github.com/hareku/its-forum-go/rc013v1"
)

func TestMessageEdit(t *testing.T) {
	message := rc013.Message{
		Header: rc013.Header{ServiceID: 1, MessageID: 1, Version: 1, VehicleID: 42},
		Common: rc013.CommonData{
			Time:              rc013.Time{Hour: 12, Minute: 34, Millisecond: 56789},
			Position:          rc013.Position{Latitude: 350000000, Longitude: 1390000000, Elevation: rc013.ElevationUnavailable},
			VehicleState:      rc013.VehicleState{Speed: 1234, Heading: 7200, SteeringWheelAngle: rc013.SteeringWheelAngleUnavailable},
			VehicleAttributes: rc013.VehicleAttributes{SizeClass: 2, RoleClass: 0, Width: 180, Length: 450},
		},
	}
	data, err := message.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := rc013.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	decoded.Common.VehicleState.Speed = 1500
	data, err = decoded.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	edited, err := rc013.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 36 || edited.Header.VehicleID != 42 || edited.Common.VehicleState.Speed != 1500 {
		t.Fatalf("unexpected edited message: length=%d, message=%+v", len(data), edited)
	}
}
