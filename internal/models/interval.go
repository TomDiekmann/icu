package models

// IntervalsResponse is the envelope returned by GET /api/v1/activity/{id}/intervals.
type IntervalsResponse struct {
	ID           string     `json:"id"`
	IcuIntervals []Interval `json:"icu_intervals"`
}

// Interval represents a single detected interval within an activity.
type Interval struct {
	Label                string  `json:"label"`
	StartIndex           int     `json:"start_index"`
	EndIndex             int     `json:"end_index"`
	Type                 string  `json:"type"`              // WORK, RECOVERY, REST, etc.
	MovingTime           int     `json:"moving_time"`       // seconds
	Distance             float64 `json:"distance"`          // meters
	AverageWatts         float64 `json:"average_watts"`
	MaxWatts             float64 `json:"max_watts"`
	WeightedAverageWatts float64 `json:"weighted_average_watts"` // normalized power
	Intensity            float64 `json:"intensity"`         // 0–100 scale (IF × 100)
	TrainingLoad         float64 `json:"training_load"`     // TSS
	AverageHR            float64 `json:"average_heartrate"`
	MaxHR                float64 `json:"max_heartrate"`
	Zone                 int     `json:"zone"`
}

// IntensityFactor returns Intensity normalised to a 0.00–1.00 scale.
func (i Interval) IntensityFactor() float64 {
	return i.Intensity / 100
}
