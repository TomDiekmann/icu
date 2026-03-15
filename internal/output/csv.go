package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
)

// PrintCSV prints a slice of structs as CSV by first converting to []map[string]interface{}
// via JSON round-trip, then using the first record's keys as headers.
func PrintCSV(data interface{}) error {
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshaling data: %w", err)
	}

	var records []map[string]interface{}
	if err := json.Unmarshal(b, &records); err != nil {
		return fmt.Errorf("unmarshaling to map: %w", err)
	}

	if len(records) == 0 {
		return nil
	}

	// collect headers from first record
	headers := make([]string, 0, len(records[0]))
	for k := range records[0] {
		headers = append(headers, k)
	}

	w := csv.NewWriter(os.Stdout)
	if err := w.Write(headers); err != nil {
		return err
	}
	for _, rec := range records {
		row := make([]string, len(headers))
		for i, h := range headers {
			row[i] = fmt.Sprintf("%v", rec[h])
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}
