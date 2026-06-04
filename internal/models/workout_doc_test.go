package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWorkoutDocText(t *testing.T) {
	tests := []struct {
		name string
		json string
		want string
	}{
		{
			name: "string form passes through",
			json: `"- 15m 55-75%\n- 2x20m 95-105%\n- 10m 55%"`,
			want: "- 15m 55-75%\n- 2x20m 95-105%\n- 10m 55%",
		},
		{
			name: "null",
			json: `null`,
			want: "",
		},
		{
			name: "object with empty steps",
			json: `{"steps":[],"locales":[],"options":{},"duration":0}`,
			want: "",
		},
		{
			name: "object with ranged and fixed power steps",
			json: `{"steps":[
				{"duration":900,"power":{"units":"%ftp","start":55,"end":75}},
				{"duration":600,"power":{"units":"%ftp","value":55}}
			],"duration":1500}`,
			want: "- 15m 55-75%\n- 10m 55%",
		},
		{
			name: "repeats render header and indented sub-steps",
			json: `{"steps":[
				{"reps":2,"steps":[
					{"duration":1200,"power":{"units":"%ftp","start":95,"end":105}},
					{"duration":300,"power":{"units":"%ftp","value":55}}
				]}
			]}`,
			want: "- 2x\n  20m 95-105%\n  5m 55%",
		},
		{
			name: "text-only step",
			json: `{"steps":[{"text":"openers as you feel"}]}`,
			want: "- openers as you feel",
		},
		{
			name: "hr step in bpm",
			json: `{"steps":[{"duration":1800,"hr":{"units":"bpm","start":120,"end":140}}]}`,
			want: "- 30m 120-140bpm",
		},
		{
			name: "warmup flag",
			json: `{"steps":[{"duration":600,"power":{"units":"%ftp","value":50},"warmup":true}]}`,
			want: "- 10m 50% warmup",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var w WorkoutDoc
			assert.NoError(t, json.Unmarshal([]byte(tt.json), &w))
			assert.Equal(t, tt.want, w.Text())
		})
	}
}

func TestEventUnmarshalWorkoutDocObject(t *testing.T) {
	// Regression: the API returns workout_doc as an object on event reads;
	// this used to fail with "cannot unmarshal object into Go struct field".
	payload := `{"id":1,"category":"WORKOUT","name":"Test",
		"workout_doc":{"steps":[{"duration":900,"power":{"units":"%ftp","start":55,"end":75}}],"duration":900}}`
	var e Event
	assert.NoError(t, json.Unmarshal([]byte(payload), &e))
	assert.Equal(t, "- 15m 55-75%", e.WorkoutDoc.Text())

	// Raw JSON round-trips losslessly through MarshalJSON.
	out, err := json.Marshal(e)
	assert.NoError(t, err)
	assert.Contains(t, string(out), `"workout_doc"`)
	assert.Contains(t, string(out), `"%ftp"`)
}

func TestWorkoutDocMarshalEmpty(t *testing.T) {
	// An event without workout_doc must omit the field, not emit null.
	e := Event{ID: 2, Category: "NOTE", Name: "rest"}
	out, err := json.Marshal(e)
	assert.NoError(t, err)
	assert.NotContains(t, string(out), "workout_doc")
}

func TestFormatStepDuration(t *testing.T) {
	tests := []struct {
		secs int
		want string
	}{
		{30, "30s"},
		{90, "1m30s"},
		{900, "15m"},
		{3600, "1h"},
		{5400, "1h30m"},
		{0, "0s"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, formatStepDuration(tt.secs), "secs=%d", tt.secs)
	}
}
