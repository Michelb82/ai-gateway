package cloudevent

import "strings"

// DefaultTypePrefix is the public CloudEvent type prefix.
const DefaultTypePrefix = "com.mywebsite.ai"

var (
	EventTypeRequest          = DefaultTypePrefix + ".request"
	EventTypeRequestCompleted = DefaultTypePrefix + ".request.completed"
	EventTypeRequestFailed    = DefaultTypePrefix + ".request.failed"
)

// ConfigureTypes sets request/completed/failed types from a prefix such as "com.mywebsite.ai".
// An empty prefix keeps the public default.
func ConfigureTypes(prefix string) {
	prefix = strings.TrimRight(strings.TrimSpace(prefix), ".")
	if prefix == "" {
		prefix = DefaultTypePrefix
	}
	EventTypeRequest = prefix + ".request"
	EventTypeRequestCompleted = prefix + ".request.completed"
	EventTypeRequestFailed = prefix + ".request.failed"
}
