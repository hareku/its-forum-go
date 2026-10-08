package consumer_test

import (
	"errors"
	"testing"

	rc013 "github.com/hareku/its-forum-go/rc013v1"
	rc016 "github.com/hareku/its-forum-go/rc016v2"
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

func TestMessageEdit(t *testing.T) {
	profile := rc016.Profile{Applications: map[uint8]rc016.ApplicationKind{0x90: rc016.ApplicationBicycle, 0x91: rc016.ApplicationPedestrian}}
	for _, application := range []rc016.Application{
		{ServiceID: 0x90, Bicycle: &rc016.BicycleData{PersonalCommon: rc016.PersonalCommon{Level: rc016.LevelUnavailable, SystemDelay: rc016.SystemDelayUnavailable}}},
		{ServiceID: 0x91, Pedestrian: &rc016.PedestrianData{PersonalCommon: rc016.PersonalCommon{Level: rc016.LevelUnavailable}, Pedestrian: rc016.Pedestrian{ItemInfo: rc016.ItemInfoUnavailable, Steps: rc016.StepsUnavailable, Motion: 3}}},
	} {
		message, err := rc016.NewMessage(rc016.Fields{Header: rc013.Header{ServiceID: 1, MessageID: 1, Version: 1}, Common: rc013.CommonData{}, Applications: []rc016.Application{application}}, profile)
		if err != nil {
			t.Fatal(err)
		}
		var header rc013.Header = message.Header
		var common rc013.CommonData = message.Common
		common.VehicleState.Speed = rc013.Speed(100)
		message.Header, message.Common = header, common
		wire, err := message.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := rc016.DecodeWithProfile(wire, profile)
		if err != nil {
			t.Fatal(err)
		}
		wantLength := 62
		if application.ServiceID == 0x90 {
			decoded.Applications[0].Bicycle.Extended.Cadence = 90
		} else {
			wantLength = 50
			decoded.Applications[0].Pedestrian.Pedestrian.Steps = 60000
		}
		wire, err = decoded.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		edited, err := rc016.DecodeWithProfile(wire, profile)
		if err != nil {
			t.Fatal(err)
		}
		if len(wire) != wantLength || edited.Header.Version != 1 || edited.Common.VehicleState.Speed != rc013.Speed(100) {
			t.Fatalf("unexpected envelope: %d %+v", len(wire), edited)
		}
		if application.ServiceID == 0x90 && edited.Applications[0].Bicycle.Extended.Cadence != 90 {
			t.Fatal("bicycle edit lost")
		}
		if application.ServiceID == 0x91 && edited.Applications[0].Pedestrian.Pedestrian.Steps != 60000 {
			t.Fatal("pedestrian edit lost")
		}
	}
	_, err := rc016.Decode(nil)
	var located *rc016.Error
	var base *rc013.Error
	if !errors.Is(err, rc016.ErrTruncated) || !errors.As(err, &located) || !errors.As(err, &base) || located != base {
		t.Fatalf("diagnostic identity: %v", err)
	}
}
