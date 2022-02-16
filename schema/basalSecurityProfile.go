package schema

import "time"

type (
	Schedule struct {
		Rate  float64 `json:"rate,omitempty"`
		Start int64   `json:"start"`
		End   int64   `json:"end"`
	}

	Profile struct {
		Type          string     `json:"type,omitempty"`
		Time          time.Time  `json:"time,omitempty"`
		Timezone      string     `json:"timezone,omitempty"`
		Guid          string     `json:"guid,omitempty"`
		BasalSchedule []Schedule `json:"basalSchedule,omitempty"`
	}
)
