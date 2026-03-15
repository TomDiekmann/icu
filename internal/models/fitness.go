package models

// FitnessDay holds fitness metrics for one day (sourced from the wellness endpoint).
type FitnessDay struct {
	Date     string  `json:"date"`
	CTL      float64 `json:"ctl"`
	ATL      float64 `json:"atl"`
	TSB      float64 `json:"tsb"`
	RampRate float64 `json:"rampRate"`
	CTLLoad  float64 `json:"ctlLoad"`
	ATLLoad  float64 `json:"atlLoad"`
}

// FormStatus returns a human-readable label and hex color for a given TSB value.
func FormStatus(tsb float64) (label string, hexColor string) {
	switch {
	case tsb > 25:
		return "VERY FRESH", "#66BB6A"
	case tsb > 5:
		return "FRESH", "#66BB6A"
	case tsb > -10:
		return "NEUTRAL", "#FFA726"
	case tsb > -30:
		return "FATIGUED", "#EF5350"
	default:
		return "OVERREACHING", "#B71C1C"
	}
}
