package models

import "encoding/json"

// WellnessEntry represents one day of wellness data from Intervals.icu.
// Field names match the API's camelCase JSON keys.
type WellnessEntry struct {
	ID       string  `json:"id"`        // YYYY-MM-DD
	CTL      float64 `json:"ctl"`       // chronic training load (fitness)
	ATL      float64 `json:"atl"`       // acute training load (fatigue)
	RampRate float64 `json:"rampRate"`  // weekly CTL ramp rate
	CTLLoad  float64 `json:"ctlLoad"`   // today's contribution to CTL
	ATLLoad  float64 `json:"atlLoad"`   // today's contribution to ATL
	Updated  string  `json:"updated"`

	// Body
	Weight        *float64 `json:"weight"`
	RestingHR     *float64 `json:"restingHR"`
	HRV           *float64 `json:"hrv"`
	HRVSDNN       *float64 `json:"hrvSDNN"`
	AvgSleepingHR *float64 `json:"avgSleepingHR"`
	SpO2          *float64 `json:"spO2"`
	BodyFat       *float64 `json:"bodyFat"`
	Abdomen       *float64 `json:"abdomen"`
	Vo2max        *float64 `json:"vo2max"`
	Systolic      *float64 `json:"systolic"`
	Diastolic     *float64 `json:"diastolic"`
	BloodGlucose  *float64 `json:"bloodGlucose"`
	Lactate       *float64 `json:"lactate"`
	Hydration     *float64 `json:"hydration"`
	HydrationVol  *float64 `json:"hydrationVolume"`
	Respiration   *float64 `json:"respiration"`
	BaevskySI     *float64 `json:"baevskySI"`

	// Sleep
	SleepSecs    *int     `json:"sleepSecs"`
	SleepScore   *float64 `json:"sleepScore"`
	SleepQuality *int     `json:"sleepQuality"`

	// Activity
	Steps *int `json:"steps"`

	// Subjective (1–10 scale)
	Soreness   *int `json:"soreness"`
	Fatigue    *int `json:"fatigue"`
	Stress     *int `json:"stress"`
	Mood       *int `json:"mood"`
	Motivation *int `json:"motivation"`
	Readiness  *int `json:"readiness"`
	Injury     *int `json:"injury"`

	// Nutrition
	KcalConsumed  *float64 `json:"kcalConsumed"`
	Carbohydrates *float64 `json:"carbohydrates"`
	Protein       *float64 `json:"protein"`
	FatTotal      *float64 `json:"fatTotal"`

	Comments *string `json:"comments"`
	Locked   *bool   `json:"locked"`

	Extra map[string]json.RawMessage `json:"-"`
}

// TSB returns the Training Stress Balance (form): CTL − ATL.
func (w WellnessEntry) TSB() float64 { return w.CTL - w.ATL }

func (w *WellnessEntry) UnmarshalJSON(data []byte) error {
	type Alias WellnessEntry
	aux := &struct{ *Alias }{Alias: (*Alias)(w)}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	w.Extra = raw
	return nil
}

func (w WellnessEntry) MarshalJSON() ([]byte, error) {
	if w.Extra != nil {
		return json.Marshal(w.Extra)
	}
	type Alias WellnessEntry
	return json.Marshal((Alias)(w))
}
