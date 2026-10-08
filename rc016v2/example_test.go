package rc016_test

import (
	"fmt"
	"github.com/hareku/its-forum-go/rc013v1"
	rc016 "github.com/hareku/its-forum-go/rc016v2"
)

func ExampleNewMessage() {
	// These service IDs belong to this experiment, not to the standard.
	profile := rc016.Profile{Applications: map[uint8]rc016.ApplicationKind{0x90: rc016.ApplicationBicycle, 0x91: rc016.ApplicationPedestrian}}
	m, err := rc016.NewMessage(rc016.Fields{
		Header: rc013.Header{ServiceID: 1, MessageID: 1, Version: 1},
		Applications: []rc016.Application{
			{ServiceID: 0x90, Bicycle: &rc016.BicycleData{PersonalCommon: rc016.PersonalCommon{Level: 7}}},
			{ServiceID: 0x91, Pedestrian: &rc016.PedestrianData{PersonalCommon: rc016.PersonalCommon{Level: 7}, Pedestrian: rc016.Pedestrian{ItemInfo: 63, Steps: 100}}},
		},
	}, profile)
	if err != nil {
		panic(err)
	}
	wire, err := m.MarshalBinary()
	if err != nil {
		panic(err)
	}
	decoded, err := rc016.DecodeWithProfile(wire, profile)
	if err != nil {
		panic(err)
	}
	decoded.Applications[1].Pedestrian.Pedestrian.Steps++
	edited, err := decoded.MarshalBinary()
	if err != nil {
		panic(err)
	}
	fmt.Println(len(edited), decoded.Header.Version, decoded.Applications[1].Pedestrian.Pedestrian.Steps)
	// Output: 75 1 101
}
