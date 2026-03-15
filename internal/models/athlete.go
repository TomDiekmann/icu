package models

import "encoding/json"

// Athlete represents an Intervals.icu athlete profile.
type Athlete struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Firstname    string   `json:"firstname"`
	Lastname     string   `json:"lastname"`
	Email        string   `json:"email"`
	Sex          string   `json:"sex"`
	City         string   `json:"city"`
	State        string   `json:"state"`
	Country      string   `json:"country"`
	Timezone     string   `json:"timezone"`
	Weight       *float64 `json:"weight"`
	IcuWeight    float64  `json:"icu_weight"`
	IcuRestingHR *float64 `json:"icu_resting_hr"`
	IcuLastSeen  string   `json:"icu_last_seen"`
	IcuActivated string   `json:"icu_activated"`

	Extra map[string]json.RawMessage `json:"-"`
}

func (a *Athlete) UnmarshalJSON(data []byte) error {
	type Alias Athlete
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

func (a Athlete) MarshalJSON() ([]byte, error) {
	if a.Extra != nil {
		return json.Marshal(a.Extra)
	}
	type Alias Athlete
	return json.Marshal((Alias)(a))
}
