package priority

import "strings"

// Level is a CloudEvent request priority lane.
type Level string

const (
	Critical Level = "CRITICAL"
	High     Level = "HIGH"
	Medium   Level = "MEDIUM"
	Low      Level = "LOW"
	None     Level = ""
)

// Parse maps a priority string to a Level. Missing or invalid values become Low.
func Parse(value string) Level {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case string(Critical):
		return Critical
	case string(High):
		return High
	case string(Medium):
		return Medium
	case string(Low):
		return Low
	default:
		return Low
	}
}

// LaneSuffix is the Redis key suffix for a priority lane.
func (l Level) LaneSuffix() string {
	switch l {
	case Critical:
		return "critical"
	case High:
		return "high"
	case Medium:
		return "medium"
	default:
		return "low"
	}
}
