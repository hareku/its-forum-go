package rc016

// Raw codes for unavailable and saturated measurements. Saturated codes denote
// the stated threshold or greater; codecs perform no physical conversion.
const (
	LevelUnavailable             uint8  = 7
	SystemDelaySaturated         uint8  = 30 // 300 ms or greater.
	SystemDelayUnavailable       uint8  = 31
	ItemInfoUnavailable          uint8  = 63
	StepsSaturated               uint16 = 65534
	StepsUnavailable             uint16 = 65535
	DrivePowerSaturated          uint8  = 254 // 2540 W or greater.
	DrivePowerUnavailable        uint8  = 255
	TireCircumferenceUnavailable uint8  = 0
	TireCircumferenceSaturated   uint8  = 255 // 2550 mm or greater.
	CadenceSaturated             uint8  = 254 // 254 rpm or greater.
	CadenceUnavailable           uint8  = 255
	GearRatioUnavailable         uint16 = 0
	GearRatioSaturated           uint16 = 1023 // 1023 percent or greater.
	TorqueSaturated              uint8  = 254  // 254 Nm or greater.
	TorqueUnavailable            uint8  = 255
	AssistPowerSaturated         uint8  = 254 // 2540 W or greater.
	AssistPowerUnavailable       uint8  = 255
	HumanPowerSaturated          uint8  = 254 // 1270 W or greater.
	HumanPowerUnavailable        uint8  = 255
	BatterySaturated             uint8  = 254 // 2540 Wh or greater.
	BatteryUnavailable           uint8  = 255
	MotionUnavailable            uint8  = 3
)
