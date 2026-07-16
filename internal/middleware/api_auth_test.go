package middleware

import "testing"

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
		ok     bool
	}{
		{name: "standard", header: "Bearer token-value", want: "token-value", ok: true},
		{name: "case insensitive scheme", header: "bearer token-value", want: "token-value", ok: true},
		{name: "surrounding whitespace", header: "  Bearer   token-value  ", want: "token-value", ok: true},
		{name: "missing token", header: "Bearer", ok: false},
		{name: "wrong scheme", header: "Basic token-value", ok: false},
		{name: "extra parts", header: "Bearer token-value extra", ok: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := extractBearerToken(test.header)
			if ok != test.ok || got != test.want {
				t.Fatalf("extractBearerToken(%q) = (%q, %v), want (%q, %v)", test.header, got, ok, test.want, test.ok)
			}
		})
	}
}
