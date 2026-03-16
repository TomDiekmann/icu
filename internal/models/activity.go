package models

import (
	"encoding/json"
	"fmt"
)

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

// zoneTimes parses icu_zone_times / icu_hr_zone_times which the API returns as
// either an array of objects [{id,secs},...] or a plain number array [12,34,...].
func zoneTimes(raw json.RawMessage, zonePrefix string) []ZoneTime {
	if len(raw) == 0 {
		return nil
	}
	// Try object array first.
	var objs []ZoneTime
	if err := json.Unmarshal(raw, &objs); err == nil {
		return objs
	}
	// Fall back to number array.
	var nums []float64
	if err := json.Unmarshal(raw, &nums); err != nil {
		return nil
	}
	out := make([]ZoneTime, len(nums))
	for i, s := range nums {
		out[i] = ZoneTime{ID: fmt.Sprintf("%s%d", zonePrefix, i+1), Secs: s}
	}
	return out
}

// UnmarshalJSON populates typed fields and captures every raw field into Extra
// so no data is lost when re-serialised in agent mode.
func (a *Activity) UnmarshalJSON(data []byte) error {
	// Use an alias without zone slice fields to avoid the recursive call and
	// to let us handle those fields manually (API returns them as number arrays).
	type Alias struct {
		ID                 string  `json:"id"`
		StartDateLocal     string  `json:"start_date_local"`
		Type               string  `json:"type"`
		Name               string  `json:"name"`
		MovingTime         int     `json:"moving_time"`
		Distance           float64 `json:"distance"`
		TotalElevationGain float64 `json:"total_elevation_gain"`
		AverageSpeed       float64 `json:"average_speed"`
		MaxSpeed           float64 `json:"max_speed"`
		IcuAverageWatts    float64 `json:"icu_average_watts"`
		IcuWeightedWatts   float64 `json:"icu_weighted_avg_watts"`
		MaxWatts           float64 `json:"max_watts"`
		AverageHeartrate   float64 `json:"average_heartrate"`
		MaxHeartrate       float64 `json:"max_heartrate"`
		IcuTrainingLoad    float64 `json:"icu_training_load"`
		IcuIntensity       float64 `json:"icu_intensity"`
		Calories           float64 `json:"calories"`
		AthleteID          string  `json:"athlete_id"`
	}
	var alias Alias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	a.ID = alias.ID
	a.StartDateLocal = alias.StartDateLocal
	a.Type = alias.Type
	a.Name = alias.Name
	a.MovingTime = alias.MovingTime
	a.Distance = alias.Distance
	a.TotalElevationGain = alias.TotalElevationGain
	a.AverageSpeed = alias.AverageSpeed
	a.MaxSpeed = alias.MaxSpeed
	a.IcuAverageWatts = alias.IcuAverageWatts
	a.IcuWeightedWatts = alias.IcuWeightedWatts
	a.MaxWatts = alias.MaxWatts
	a.AverageHeartrate = alias.AverageHeartrate
	a.MaxHeartrate = alias.MaxHeartrate
	a.IcuTrainingLoad = alias.IcuTrainingLoad
	a.IcuIntensity = alias.IcuIntensity
	a.Calories = alias.Calories
	a.AthleteID = alias.AthleteID

	// Parse zone fields manually — API returns these as either [{id,secs}] or [num,num,...].
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if r, ok := raw["icu_zone_times"]; ok {
		a.IcuZoneTimes = zoneTimes(r, "Z")
	}
	if r, ok := raw["icu_hr_zone_times"]; ok {
		a.IcuHRZoneTimes = zoneTimes(r, "Z")
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
