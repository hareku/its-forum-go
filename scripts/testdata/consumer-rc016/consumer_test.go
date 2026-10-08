package consumer_test

import (
	"errors"
	"testing"

	rc013 "github.com/hareku/its-forum-go/rc013v1"
	rc016 "github.com/hareku/its-forum-go/rc016v1"
)

func TestPublicTypeIdentity(t *testing.T) {
	var issue rc016.Issue = rc013.Issue{Field: "identity", Message: "shared type"}
	var original rc013.Issue = issue
	var located *rc016.Error = &rc013.Error{Field: original.Field, Offset: -1, Err: rc013.ErrLayout}
	var base *rc013.Error = located
	if !errors.Is(base, rc016.ErrLayout) || rc016.ErrLayout != rc013.ErrLayout ||
		rc016.ErrTruncated != rc013.ErrTruncated || rc016.ErrMalformed != rc013.ErrMalformed ||
		rc016.ErrRange != rc013.ErrRange || rc016.ErrUnsupported != rc013.ErrUnsupported {
		t.Fatal("PUBLIC_TYPE_IDENTITY: diagnostic identity changed")
	}
}

func TestBicycleEdit(t *testing.T) {
	// The application mapping is specific to this example's experiment.
	profile := rc016.BicycleProfile{Applications: map[uint8]rc016.BicycleApplicationKind{0x90: rc016.ApplicationPersonalCommon}}
	message, err := rc016.NewBicycle(rc016.BicycleFields{
		Header:       rc013.Header{ServiceID: 1, MessageID: 1, Version: 1},
		Applications: []rc016.BicycleApplication{{ServiceID: 0x90, PersonalCommon: &rc016.PersonalCommon{Level: 5, SystemDelay: 3}}},
	}, profile)
	if err != nil {
		t.Fatal(err)
	}
	data, err := message.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := rc016.DecodeBicycleWithProfile(data, profile)
	if err != nil {
		t.Fatal(err)
	}
	decoded.Applications[0].PersonalCommon.SystemDelay = 4
	data, err = decoded.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	edited, err := rc016.DecodeBicycleWithProfile(data, profile)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 45 || edited.Applications[0].PersonalCommon.SystemDelay != 4 {
		t.Fatalf("unexpected bicycle edit: length=%d, message=%+v", len(data), edited)
	}
}

func TestRoadsideEdit(t *testing.T) {
	message, err := rc016.NewRoadside(rc016.RoadsideFields{
		Header: rc016.RoadsideHeader{Version: 1},
		Groups: []rc016.RoadsideObjectGroup{{SensorIndex: -1, Objects: []rc016.RoadsideObject{{MessageID: 1, Version: 1, ID: 7}}}},
	}, rc016.RoadsideProfile{})
	if err != nil {
		t.Fatal(err)
	}
	data, err := message.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := rc016.DecodeRoadside(data)
	if err != nil {
		t.Fatal(err)
	}
	decoded.Groups[0].Objects[0].Common.VehicleState.Speed = rc013.Speed(100)
	data, err = decoded.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	edited, err := rc016.DecodeRoadside(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 55 || !edited.ObjectsDecoded() || edited.Groups[0].Objects[0].ID != 7 || edited.Groups[0].Objects[0].Common.VehicleState.Speed != 100 {
		t.Fatalf("unexpected roadside edit: length=%d, message=%+v", len(data), edited)
	}
}

func TestCSMAEdit(t *testing.T) {
	message := rc016.CSMAMessage{Header: rc016.CSMAHeader{Version: 1}, Objects: []rc016.CSMAObject{{ID: 7, Type: 4, Size: 2}}}
	data, err := message.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := rc016.DecodeCSMA(data)
	if err != nil {
		t.Fatal(err)
	}
	decoded.Objects[0].Speed = 1234
	data, err = decoded.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	edited, err := rc016.DecodeCSMA(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 36 || edited.Objects[0].Speed != 1234 {
		t.Fatalf("unexpected CSMA edit: length=%d, message=%+v", len(data), edited)
	}
}
