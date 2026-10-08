package rc016

// BicycleData is one complete 22-byte bicycle application payload.
type BicycleData struct {
	PersonalCommon PersonalCommon
	Basic          BicycleBasic
	Extended       BicycleExtended
}

// PedestrianData is one complete 10-byte pedestrian application payload.
type PedestrianData struct {
	PersonalCommon PersonalCommon
	Pedestrian     Pedestrian
}

// MarshalBinary encodes all three frames in specification order.
func (v BicycleData) MarshalBinary() ([]byte, error) {
	common, err := v.PersonalCommon.MarshalBinary()
	if err != nil {
		return nil, err
	}
	basic, err := v.Basic.MarshalBinary()
	if err != nil {
		return nil, rebaseError(err, 5)
	}
	extended, err := v.Extended.MarshalBinary()
	if err != nil {
		return nil, rebaseError(err, 8)
	}
	return append(append(common, basic...), extended...), nil
}

// UnmarshalBinary decodes exactly 22 bytes and leaves the receiver unchanged on error.
func (v *BicycleData) UnmarshalBinary(data []byte) error {
	if v == nil {
		return &Error{Field: "BicycleData", Offset: -1, Err: ErrMalformed}
	}
	if err := payloadSize(data, 22, "BicycleData"); err != nil {
		return err
	}
	var decoded BicycleData
	if err := decoded.PersonalCommon.UnmarshalBinary(data[:5]); err != nil {
		return err
	}
	if err := decoded.Basic.UnmarshalBinary(data[5:8]); err != nil {
		return rebaseError(err, 5)
	}
	if err := decoded.Extended.UnmarshalBinary(data[8:]); err != nil {
		return rebaseError(err, 8)
	}
	*v = decoded
	return nil
}

// MarshalBinary encodes both frames in specification order.
func (v PedestrianData) MarshalBinary() ([]byte, error) {
	common, err := v.PersonalCommon.MarshalBinary()
	if err != nil {
		return nil, err
	}
	pedestrian, err := v.Pedestrian.MarshalBinary()
	if err != nil {
		return nil, rebaseError(err, 5)
	}
	return append(common, pedestrian...), nil
}

// UnmarshalBinary decodes exactly 10 bytes and leaves the receiver unchanged on error.
func (v *PedestrianData) UnmarshalBinary(data []byte) error {
	if v == nil {
		return &Error{Field: "PedestrianData", Offset: -1, Err: ErrMalformed}
	}
	if err := payloadSize(data, 10, "PedestrianData"); err != nil {
		return err
	}
	var decoded PedestrianData
	if err := decoded.PersonalCommon.UnmarshalBinary(data[:5]); err != nil {
		return err
	}
	if err := decoded.Pedestrian.UnmarshalBinary(data[5:]); err != nil {
		return rebaseError(err, 5)
	}
	*v = decoded
	return nil
}

func payloadSize(data []byte, size int, field string) error {
	if len(data) == size {
		return nil
	}
	kind := ErrMalformed
	if len(data) < size {
		kind = ErrTruncated
	}
	return &Error{Field: field, Offset: len(data), Err: kind}
}
