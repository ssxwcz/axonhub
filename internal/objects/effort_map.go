package objects

import (
	"encoding/json"
	"errors"
	"io"
)

// EffortMap maps an incoming reasoning effort to the value actually sent
// upstream, following cc-switch's thinkingLevelMap philosophy:
//   - string value -> the effort sent to the provider (rename or downgrade,
//     e.g. "max" -> "xhigh" when the model caps below max)
//   - null value   -> that effort is explicitly unsupported: reasoning is
//     cleared so nothing reaches the provider for it
//   - missing key  -> pass through unchanged
//
// It is a named map type so it can be bound to a gqlgen custom scalar while
// keeping the sparse map[string]*string representation the rest of the code
// operates on. Marshal/UnmarshalGQL round-trip the null-pointer "disabled"
// semantics through the GraphQL wire format.
type EffortMap map[string]*string

// MarshalGQL writes the map as a JSON object, preserving nil pointers as null.
func (m EffortMap) MarshalGQL(w io.Writer) {
	if m == nil {
		_, _ = w.Write([]byte("null"))
		return
	}

	b, err := json.Marshal(map[string]*string(m))
	if err != nil {
		// A map of string->*string always marshals; fall back to null defensively.
		_, _ = w.Write([]byte("null"))
		return
	}

	_, _ = w.Write(b)
}

// UnmarshalGQL accepts the JSON object form back into the sparse map.
// gqlgen hands the decoded scalar value as a map[string]any (or nil).
func (m *EffortMap) UnmarshalGQL(v any) error {
	if m == nil {
		return errors.New("EffortMap: UnmarshalGQL on nil pointer")
	}

	if v == nil {
		*m = nil
		return nil
	}

	switch in := v.(type) {
	case EffortMap:
		*m = in
		return nil
	case map[string]any:
		out := make(EffortMap, len(in))
		for key, val := range in {
			if val == nil {
				// nil pointer = "this effort is disabled".
				out[key] = nil
				continue
			}
			s, ok := val.(string)
			if !ok {
				// Non-string values are not valid effort overrides; surface them to
				// NormalizeModelConfigs as a clear validation error rather than
				// silently coercing.
				b, _ := json.Marshal(val)
				return errors.New("effortMap value for " + key + " must be a string or null, got " + string(b))
			}
			clone := s
			out[key] = &clone
		}
		*m = out
		return nil
	}

	// Fallback: round-trip through JSON for any other shape gqlgen may pass.
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	var mm map[string]*string
	if err := json.Unmarshal(b, &mm); err != nil {
		return err
	}
	*m = EffortMap(mm)
	return nil
}
