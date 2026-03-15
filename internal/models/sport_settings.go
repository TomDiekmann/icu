package models

// SportSettings holds FTP, zone boundaries, and HR parameters for a group of sport types.
// Retrieved from GET /api/v1/athlete/{id}/sport-settings.
type SportSettings struct {
	ID             int      `json:"id"`
	AthleteID      string   `json:"athlete_id"`
	Types          []string `json:"types"`
	FTP            *float64 `json:"ftp"`
	IndoorFTP      *float64 `json:"indoor_ftp"`
	LTHR           *float64 `json:"lthr"`
	MaxHR          *float64 `json:"max_hr"`
	WPrime         *float64 `json:"w_prime"`
	SweetSpotMin   float64  `json:"sweet_spot_min"`
	SweetSpotMax   float64  `json:"sweet_spot_max"`
	WarmupTime     int      `json:"warmup_time"`
	CooldownTime   int      `json:"cooldown_time"`

	// Power zones: upper boundaries as % of FTP. Last value is 999 (sentinel for "max").
	PowerZones     []float64 `json:"power_zones"`
	PowerZoneNames []string  `json:"power_zone_names"`

	// HR zones: upper boundaries in absolute BPM.
	HRZones     []float64 `json:"hr_zones"`
	HRZoneNames []string  `json:"hr_zone_names"`

	// Pace zones: upper boundaries in m/s (nil when not configured).
	PaceZones     []float64 `json:"pace_zones"`
	PaceZoneNames []string  `json:"pace_zone_names"`
	ThresholdPace *float64  `json:"threshold_pace"`
}
