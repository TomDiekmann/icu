package models

import "encoding/json"

// Activity represents an Intervals.icu activity.
// Field names and types match the actual API response (verified against the API).
// All raw fields are preserved in Extra for lossless JSON pass-through in agent mode.
type Activity struct {
	ID                 string  `json:"id"`                   // e.g. "i132173665"
	StartDateLocal     string  `json:"start_date_local"`
	Type               string  `json:"type"`
	Name               string  `json:"name"`
	MovingTime         int     `json:"moving_time"`          // seconds
	Distance           float64 `json:"distance"`             // meters
	TotalElevationGain float64 `json:"total_elevation_gain"` // meters
	AverageSpeed       float64 `json:"average_speed"`        // m/s
	MaxSpeed           float64 `json:"max_speed"`            // m/s
	IcuAverageWatts    float64 `json:"icu_average_watts"`
	IcuWeightedWatts   float64 `json:"icu_weighted_avg_watts"`
	MaxWatts           float64 `json:"max_watts"`
	AverageHeartrate   float64 `json:"average_heartrate"`
	MaxHeartrate       float64 `json:"max_heartrate"`
	IcuTrainingLoad    float64 `json:"icu_training_load"`    // TSS
	IcuIntensity       float64 `json:"icu_intensity"`        // IF (0–100 scale, divide by 100)
	Calories           float64 `json:"calories"`
	AthleteID          string  `json:"athlete_id"`

	// Zone time distribution (populated in activity detail).
	// Each entry has an ID like "Z1"–"Z7" or "SS" and time in seconds.
	IcuZoneTimes   []ZoneTime `json:"icu_zone_times"`
	IcuHRZoneTimes []ZoneTime `json:"icu_hr_zone_times"`

	Extra map[string]json.RawMessage `json:"-"`
}

// ZoneTime holds the time spent in a single zone.
type ZoneTime struct {
	ID   string  `json:"id"`
	Secs float64 `json:"secs"`
}

// UnmarshalJSON populates typed fields and captures every raw field into Extra
// so no data is lost when re-serialised in agent mode.
func (a *Activity) UnmarshalJSON(data []byte) error {
	type Alias Activity
	aux := &struct{ *Alias }{Alias: (*Alias)(a)}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	a.Extra = raw
	return nil
}

// MarshalJSON outputs the full raw map so agents receive every API field.
func (a Activity) MarshalJSON() ([]byte, error) {
	if a.Extra != nil {
		return json.Marshal(a.Extra)
	}
	type Alias Activity
	return json.Marshal((Alias)(a))
}

// IntensityFactor returns IcuIntensity normalised to a 0.00–1.00 scale.
// The API returns IF × 100 (e.g. 83.4 means IF 0.834).
func (a Activity) IntensityFactor() float64 {
	return a.IcuIntensity / 100
}
