package rc016

// Validate reports unassigned codes without modifying the frame.
func (v PersonalCommon) Validate() []Issue {
	var issues []Issue
	if v.Level < 1 || (v.Level > 5 && v.Level != LevelUnavailable) {
		issues = append(issues, Issue{Field: "PersonalCommon.Level", Message: "unassigned equipment level"})
	}
	return issues
}

// Validate reports reserved codes; collision codes 1..15 remain unspecified TBD.
func (v BicycleBasic) Validate() []Issue {
	var issues []Issue
	if v.AssistType > 2 {
		issues = append(issues, Issue{Field: "BicycleBasic.AssistType", Message: "reserved assist type"})
	}
	if v.BicycleType > 7 {
		issues = append(issues, Issue{Field: "BicycleBasic.BicycleType", Message: "reserved bicycle type"})
	}
	if v.Pedaling > 2 {
		issues = append(issues, Issue{Field: "BicycleBasic.Pedaling", Message: "reserved pedaling state"})
	}
	return issues
}

// Validate reports reserved state codes and nonzero reserved bits.
func (v BicycleExtended) Validate() []Issue {
	var issues []Issue
	if v.RearLight > 2 {
		issues = append(issues, Issue{Field: "BicycleExtended.RearLight", Message: "reserved rear light state"})
	}
	if v.DriveUnitStatus > 2 {
		issues = append(issues, Issue{Field: "BicycleExtended.DriveUnitStatus", Message: "reserved drive unit state"})
	}
	if v.Maintenance > 2 {
		issues = append(issues, Issue{Field: "BicycleExtended.Maintenance", Message: "reserved maintenance state"})
	}
	if v.Reserved != 0 {
		issues = append(issues, Issue{Field: "BicycleExtended.Reserved", Message: "nonzero reserved bits"})
	}
	return issues
}

// Validate reports reserved item codes and nonzero reserved bits. The reserved
// bits diagnostic is library policy, not an explicit transmit-zero requirement.
func (v Pedestrian) Validate() []Issue {
	var issues []Issue
	if v.ItemInfo != 1 && v.ItemInfo != 2 && v.ItemInfo != ItemInfoUnavailable {
		issues = append(issues, Issue{Field: "Pedestrian.ItemInfo", Message: "reserved item information"})
	}
	if v.Reserved != 0 {
		issues = append(issues, Issue{Field: "Pedestrian.Reserved", Message: "nonzero reserved bits"})
	}
	return issues
}

// Validate reports the semantic concerns of all constituent frames.
func (v BicycleData) Validate() []Issue {
	issues := v.PersonalCommon.Validate()
	issues = append(issues, v.Basic.Validate()...)
	return append(issues, v.Extended.Validate()...)
}

// Validate reports the semantic concerns of both constituent frames.
func (v PedestrianData) Validate() []Issue {
	return append(v.PersonalCommon.Validate(), v.Pedestrian.Validate()...)
}
