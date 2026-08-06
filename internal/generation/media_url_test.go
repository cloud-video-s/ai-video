package generation

import (
	"reflect"
	"testing"

	"ai-video/internal/config"
	"ai-video/internal/pkg/upload"
)

func TestProviderGenerationInputAcceptsHalfAndFullMediaURLs(t *testing.T) {
	previousUpload := config.Cfg.Upload
	previousDB := config.DB
	config.DB = nil
	config.Cfg.Upload = config.UploadConfig{
		StorageProvider: upload.StorageAliyunOSS,
		OSSBaseURL:      "https://origin.example.com/",
		ProxyBaseURL:    "https://cdn.example.com/",
	}
	t.Cleanup(func() {
		config.Cfg.Upload = previousUpload
		config.DB = previousDB
	})

	tests := []struct {
		name  string
		input map[string]any
		want  map[string]any
	}{
		{
			name: "reference images",
			input: map[string]any{
				"prompt": "animate",
				"images": []string{
					"/uploads/images/half.png",
					"https://cdn.example.com/uploads/images/full.png",
					"https://external.example.com/reference.png",
				},
			},
			want: map[string]any{
				"prompt": "animate",
				"images": []string{
					"https://cdn.example.com/uploads/images/half.png",
					"https://cdn.example.com/uploads/images/full.png",
					"https://external.example.com/reference.png",
				},
			},
		},
		{
			name: "reference video half URL",
			input: map[string]any{
				"prompt": "restyle",
				"video":  "/uploads/videos/reference.mp4",
			},
			want: map[string]any{
				"prompt": "restyle",
				"video":  "https://cdn.example.com/uploads/videos/reference.mp4",
			},
		},
		{
			name: "reference video full URL",
			input: map[string]any{
				"prompt": "restyle",
				"video":  "https://cdn.example.com/uploads/videos/reference.mp4",
			},
			want: map[string]any{
				"prompt": "restyle",
				"video":  "https://cdn.example.com/uploads/videos/reference.mp4",
			},
		},
		{
			name: "first and end frames",
			input: map[string]any{
				"prompt":      "transition",
				"first_frame": "/uploads/images/first.png",
				"end_frame":   "https://cdn.example.com/uploads/images/end.png",
			},
			want: map[string]any{
				"prompt":      "transition",
				"first_frame": "https://cdn.example.com/uploads/images/first.png",
				"end_frame":   "https://cdn.example.com/uploads/images/end.png",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := providerGenerationInput(test.input); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("provider input = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestNormalizeOwnedGenerationInputAcceptsHalfAndFullMediaURLs(t *testing.T) {
	input := GenerationInput{
		Images: []string{
			"/uploads/images/half.png",
			"https://cdn.example.com/uploads/images/full.png?preview=1",
			"https://external.example.com/reference.png",
		},
		Video:      "https://cdn.example.com/uploads/videos/reference.mp4",
		FirstFrame: "/uploads/images/first.png",
		EndFrame:   "https://cdn.example.com/uploads/images/end.png",
	}
	source := map[string]any{}
	owned := map[string]struct{}{
		"/uploads/images/half.png":      {},
		"/uploads/images/full.png":      {},
		"/uploads/videos/reference.mp4": {},
		"/uploads/images/first.png":     {},
		"/uploads/images/end.png":       {},
	}

	normalizeOwnedGenerationInput(source, input, owned)

	want := map[string]any{
		"images": []string{
			"/uploads/images/half.png",
			"/uploads/images/full.png",
			"https://external.example.com/reference.png",
		},
		"video":       "/uploads/videos/reference.mp4",
		"first_frame": "/uploads/images/first.png",
		"end_frame":   "/uploads/images/end.png",
	}
	if !reflect.DeepEqual(source, want) {
		t.Fatalf("normalized input = %#v, want %#v", source, want)
	}
}
