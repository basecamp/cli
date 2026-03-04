package output

import (
	"bytes"
	"encoding/json"
)

// NormalizeData converts json.RawMessage and other types to standard Go types.
func NormalizeData(data any) any {
	// Handle json.RawMessage by unmarshaling it
	if raw, ok := data.(json.RawMessage); ok {
		var unmarshaled any
		if err := unmarshalPreservingNumbers(raw, &unmarshaled); err == nil {
			return normalizeUnmarshaled(unmarshaled)
		}
		return data
	}

	// Handle typed structs/slices by marshaling then unmarshaling
	// This converts struct types to map[string]any
	switch data.(type) {
	case []map[string]any, map[string]any, []any:
		return data // Already normalized
	case nil:
		return data
	default:
		// Try to convert via JSON round-trip
		b, err := json.Marshal(data)
		if err != nil {
			return data
		}
		var unmarshaled any
		if err := unmarshalPreservingNumbers(b, &unmarshaled); err != nil {
			return data
		}
		return normalizeUnmarshaled(unmarshaled)
	}
}

// unmarshalPreservingNumbers decodes JSON using UseNumber so numeric values
// remain as json.Number instead of being converted to float64. This preserves
// precision for large integer IDs that exceed 53-bit float64 range.
func unmarshalPreservingNumbers(data []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	return dec.Decode(v)
}

// normalizeUnmarshaled converts []any to []map[string]any if all elements are maps.
func normalizeUnmarshaled(v any) any {
	switch d := v.(type) {
	case []any:
		// Check if all elements are maps, convert to []map[string]any
		if len(d) == 0 {
			return []map[string]any{}
		}
		maps := make([]map[string]any, 0, len(d))
		for _, item := range d {
			if m, ok := item.(map[string]any); ok {
				maps = append(maps, m)
			} else {
				return v // Mixed types, return as-is
			}
		}
		return maps
	default:
		return v
	}
}
