package upload

import "testing"

func TestHalfURL(t *testing.T) {
	tests := map[string]string{
		"https://cdn.example.com/uploads/images/a.png?token=secret#preview": "/uploads/images/a.png",
		"uploads/generated/19/result.mp4":                                   "/uploads/generated/19/result.mp4",
		"/uploads/images/a.png":                                             "/uploads/images/a.png",
	}
	for input, want := range tests {
		if got := HalfURL(input); got != want {
			t.Fatalf("HalfURL(%q) = %q, want %q", input, got, want)
		}
	}
}
