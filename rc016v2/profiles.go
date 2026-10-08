package rc016

// ApplicationKind identifies a complete RC-016 payload, not a wire service ID.
type ApplicationKind uint8

const (
	ApplicationBicycle    ApplicationKind = 1
	ApplicationPedestrian ApplicationKind = 2
)

// Profile maps experiment-assigned service IDs to complete application payloads.
// DecodeWithProfile and NewMessage take a private snapshot of the map.
type Profile struct{ Applications map[uint8]ApplicationKind }

func snapshotProfile(profile Profile) (Profile, error) {
	frozen := Profile{Applications: make(map[uint8]ApplicationKind, len(profile.Applications))}
	for id, kind := range profile.Applications {
		if kind < ApplicationBicycle || kind > ApplicationPedestrian {
			return Profile{}, &Error{Field: "message.profile", Offset: -1, Err: ErrUnsupported}
		}
		frozen.Applications[id] = kind
	}
	return frozen, nil
}
