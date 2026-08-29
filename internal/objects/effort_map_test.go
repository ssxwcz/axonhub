package objects

import (
	"bytes"
	"encoding/json"
	"testing"
)

// EffortMap is a gqlgen custom scalar bound to a named map type. Its
// Marshal/UnmarshalGQL pair must round-trip the null-pointer "this effort is
// disabled" semantics, otherwise saving a per-model effort override would
// silently drop the disabled entries on the next edit.

func TestEffortMap_MarshalGQL(t *testing.T) {
	high := "high"
	tests := []struct {
		name string
		m     EffortMap
		want string
	}{
		{name: "nil map", m: nil, want: "null"},
		{name: "empty map", m: EffortMap{}, want: "{}"},
		{
			name: "string value and disabled entry",
			m:    EffortMap{"max": nil, "low": &high},
			want: `{"low":"high","max":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tt.m.MarshalGQL(&buf)
			// Order-independent compare by re-parsing.
			var got, want any
			if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
				t.Fatalf("marshal produced invalid JSON %q: %v", buf.String(), err)
			}
			if err := json.Unmarshal([]byte(tt.want), &want); err != nil {
				t.Fatalf("want %q is invalid JSON: %v", tt.want, err)
			}
			if !equalJSON(got, want) {
				t.Errorf("marshal = %s, want %s", buf.String(), tt.want)
			}
		})
	}
}

func TestEffortMap_UnmarshalGQL_RoundTrip(t *testing.T) {
	high := "high"
	src := EffortMap{"max": nil, "low": &high}

	var buf bytes.Buffer
	src.MarshalGQL(&buf)

	var decoded EffortMap
	if err := decoded.UnmarshalGQL(decodeToAny(t, buf.Bytes())); err != nil {
		t.Fatalf("UnmarshalGQL: %v", err)
	}

	if len(decoded) != len(src) {
		t.Fatalf("len = %d, want %d", len(decoded), len(src))
	}
	// nil pointer must survive the round trip: it is the "disabled" signal.
	if decoded["max"] != nil {
		t.Errorf("max = %v, want nil (disabled)", ptrVal(decoded["max"]))
	}
	if got := ptrVal(decoded["low"]); got != "high" {
		t.Errorf("low = %q, want high", got)
	}
}

func TestEffortMap_UnmarshalGQL_NilInput(t *testing.T) {
	var m EffortMap
	if err := m.UnmarshalGQL(nil); err != nil {
		t.Fatalf("nil input: %v", err)
	}
	if m != nil {
		t.Errorf("expected nil map, got %v", m)
	}
}

func TestEffortMap_UnmarshalGQL_RejectsNonStringValue(t *testing.T) {
	var m EffortMap
	in := map[string]any{"max": 31999}
	if err := m.UnmarshalGQL(in); err == nil {
		t.Fatal("expected error for non-string effort value, got nil")
	}
}

func ptrVal(p *string) string {
	if p == nil {
		return "<nil>"
	}
	return *p
}

func decodeToAny(t *testing.T, b []byte) any {
	t.Helper()
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatalf("invalid JSON %s: %v", b, err)
	}
	return v
}

func equalJSON(a, b any) bool {
	switch av := a.(type) {
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for k, v := range av {
			if !equalJSON(v, bv[k]) {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}
