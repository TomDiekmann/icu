package output

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"time"
)

type Meta struct {
	Command   string            `json:"command"`
	AthleteID string            `json:"athlete_id"`
	Timestamp string            `json:"timestamp"`
	Count     int               `json:"count,omitempty"`
	Filters   map[string]string `json:"filters,omitempty"`
}

type Response struct {
	Meta Meta        `json:"meta"`
	Data interface{} `json:"data"`
}

func PrintJSON(command, athleteID string, filters map[string]string, data interface{}) error {
	count := 0
	if v := reflect.ValueOf(data); v.IsValid() && (v.Kind() == reflect.Slice || v.Kind() == reflect.Array) {
		count = v.Len()
	}
	resp := Response{
		Meta: Meta{
			Command:   command,
			AthleteID: athleteID,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Count:     count,
			Filters:   filters,
		},
		Data: data,
	}
	b, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}
	fmt.Fprintln(os.Stdout, string(b))
	return nil
}

func PrintError(err error, exitCode int) {
	msg := map[string]interface{}{
		"error": err.Error(),
		"code":  exitCode,
	}
	b, _ := json.MarshalIndent(msg, "", "  ")
	fmt.Fprintln(os.Stderr, string(b))
}
