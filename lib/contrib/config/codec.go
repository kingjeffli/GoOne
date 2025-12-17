package config

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Marshal serializes v into bytes according to format.
// Supported: json, yaml/yml. Unknown formats fallback to json.
func Marshal(format string, v any) ([]byte, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "json":
		return json.Marshal(v)
	case "yaml", "yml":
		return yaml.Marshal(v)
	default:
		// keep behaviour forgiving: fallback to json to avoid hard failure.
		b, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("marshal fallback(json) for format=%q: %w", format, err)
		}
		return b, nil
	}
}


