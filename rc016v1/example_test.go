package rc016_test

import (
	"fmt"

	"github.com/hareku/its-forum-go/rc013v1"
	rc016 "github.com/hareku/its-forum-go/rc016v1"
)

func ExampleNewBicycle() {
	// Application ID 0x90 belongs to this experiment, not a universal assignment.
	profile := rc016.BicycleProfile{Applications: map[uint8]rc016.BicycleApplicationKind{0x90: rc016.ApplicationPersonalCommon}}
	message, err := rc016.NewBicycle(rc016.BicycleFields{
		Header:       rc013.Header{ServiceID: 1, MessageID: 1, Version: 1},
		Applications: []rc016.BicycleApplication{{ServiceID: 0x90, PersonalCommon: &rc016.PersonalCommon{Level: 5, SystemDelay: 3}}},
	}, profile)
	if err != nil {
		panic(err)
	}
	data, err := message.MarshalBinary()
	if err != nil {
		panic(err)
	}
	decoded, err := rc016.DecodeBicycleWithProfile(data, profile)
	if err != nil {
		panic(err)
	}
	decoded.Applications[0].PersonalCommon.SystemDelay = 4
	edited, err := decoded.MarshalBinary()
	if err != nil {
		panic(err)
	}
	fmt.Println(len(edited), decoded.Applications[0].PersonalCommon.SystemDelay)
	// Output: 45 4
}

func ExampleDecodeBicycle() {
	message, err := rc016.NewBicycle(rc016.BicycleFields{
		Header:       rc013.Header{ServiceID: 1, MessageID: 1, Version: 1},
		Applications: []rc016.BicycleApplication{{ServiceID: 0xee, Data: []byte{0xde, 0xad}}},
	}, rc016.BicycleProfile{})
	if err != nil {
		panic(err)
	}
	data, err := message.MarshalBinary()
	if err != nil {
		panic(err)
	}
	decoded, err := rc016.DecodeBicycle(data)
	if err != nil {
		panic(err)
	}
	// Without an experiment mapping, payload bytes remain explicitly opaque.
	fmt.Printf("%x\n", decoded.Applications[0].Data)
	// Output: dead
}

func ExamplePedestrian() {
	fields := rc016.Pedestrian{ShoeAttribute: 3, Steps: 100, Motion: 1}
	data, err := fields.MarshalBinary()
	if err != nil {
		panic(err)
	}
	var decoded rc016.Pedestrian
	if err := decoded.UnmarshalBinary(data); err != nil {
		panic(err)
	}
	fmt.Println(len(data), decoded.Steps)
	// Output: 5 100
}

func ExampleCSMAMessage_MarshalBinary() {
	message := rc016.CSMAMessage{Header: rc016.CSMAHeader{Version: 1}, Objects: []rc016.CSMAObject{{ID: 7, Type: 4, Size: 2}}}
	data, err := message.MarshalBinary()
	if err != nil {
		panic(err)
	}
	decoded, err := rc016.DecodeCSMA(data)
	if err != nil {
		panic(err)
	}
	decoded.Objects[0].Speed = 1234
	data, err = decoded.MarshalBinary()
	if err != nil {
		panic(err)
	}
	fmt.Println(len(data), decoded.Objects[0].Speed)
	// Output: 36 1234
}

func ExampleNewRoadside() {
	message, err := rc016.NewRoadside(rc016.RoadsideFields{
		Header: rc016.RoadsideHeader{Version: 1},
		Groups: []rc016.RoadsideObjectGroup{{SensorIndex: -1, Objects: []rc016.RoadsideObject{{MessageID: 1, Version: 1, ID: 7}}}},
	}, rc016.RoadsideProfile{})
	if err != nil {
		panic(err)
	}
	data, err := message.MarshalBinary()
	if err != nil {
		panic(err)
	}
	decoded, err := rc016.DecodeRoadside(data)
	if err != nil {
		panic(err)
	}
	decoded.Groups[0].Objects[0].Common.VehicleState.Speed = 100
	data, err = decoded.MarshalBinary()
	if err != nil {
		panic(err)
	}
	fmt.Println(len(data), decoded.ObjectsDecoded(), decoded.Groups[0].Objects[0].ID)
	// Output: 55 true 7
}
