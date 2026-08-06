package upload

import "testing"

func TestHTTPHandlerExpandsResponseURLWithoutChangingPersistedValue(t *testing.T) {
	handler := NewHTTPHandler(nil, WithPublicURLResolver(func(value string) string {
		return "https://test-cdn.zdrawai.com" + value
	}))
	session := &Session{FileURL: "/uploads/images/example.png"}
	responseSession := handler.responseSession(session)
	if responseSession.FileURL != "/uploads/images/example.png" {
		t.Fatalf("response file URL = %q, want half URL", responseSession.FileURL)
	}
	if responseSession.PreviewURL != "https://test-cdn.zdrawai.com/uploads/images/example.png" {
		t.Fatalf("response preview URL = %q", responseSession.PreviewURL)
	}
	if session.FileURL != "/uploads/images/example.png" {
		t.Fatalf("persisted session URL was changed to %q", session.FileURL)
	}

	credential := &DirectUploadCredential{FileURL: "/uploads/videos/example.mp4"}
	responseCredential := handler.responseCredential(credential)
	if responseCredential.FileURL != "/uploads/videos/example.mp4" {
		t.Fatalf("response credential file URL = %q, want half URL", responseCredential.FileURL)
	}
	if responseCredential.PreviewURL != "https://test-cdn.zdrawai.com/uploads/videos/example.mp4" {
		t.Fatalf("response credential preview URL = %q", responseCredential.PreviewURL)
	}
	if credential.FileURL != "/uploads/videos/example.mp4" {
		t.Fatalf("persisted credential URL was changed to %q", credential.FileURL)
	}
}
