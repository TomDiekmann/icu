package models

import "encoding/json"

// Event represents a calendar event from Intervals.icu.
// Covers workouts, notes, races, rest days, and other planned items.
type Event struct {
	ID             int64    `json:"id,omitempty"`
	StartDateLocal string   `json:"start_date_local,omitempty"`
	Name           string   `json:"name,omitempty"`
	Description    string   `json:"description,omitempty"`
	Type           string   `json:"type,omitempty"`      // sport type: Ride, Run, Swim, ...
	Category       string   `json:"category,omitempty"` // WORKOUT, NOTE, RACE, REST_DAY, ...
	WorkoutDoc     string   `json:"workout_doc,omitempty"`
	LoadTarget     *float64 `json:"load_target,omitempty"`
	Duration       *int     `json:"duration,omitempty"` // target duration in seconds
	Indoor         *bool    `json:"indoor,omitempty"`
	AthleteID      string   `json:"athlete_id,omitempty"`
	Paired         *bool    `json:"paired,omitempty"`

	Extra map[string]json.RawMessage `json:"-"`
}

func (e *Event) UnmarshalJSON(data []byte) error {
	type Alias Event
	aux := &struct{ *Alias }{Alias: (*Alias)(e)}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	e.Extra = raw
	return nil
}

func (e Event) MarshalJSON() ([]byte, error) {
	if e.Extra != nil {
		return json.Marshal(e.Extra)
	}
	type Alias Event
	return json.Marshal((Alias)(e))
}
