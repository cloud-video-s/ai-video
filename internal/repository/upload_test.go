package repository

import (
	"testing"

	"ai-video/internal/domain"
	"ai-video/internal/pkg/upload"
)

func TestUploadOwnerUserType(t *testing.T) {
	tests := []struct {
		owner upload.UploaderType
		want  int8
	}{
		{owner: upload.UploaderAdmin, want: domain.UploadUserAdmin},
		{owner: upload.UploaderAPIUser, want: domain.UploadUserClient},
	}
	for _, tt := range tests {
		got, err := uploadOwnerUserType(tt.owner)
		if err != nil {
			t.Fatalf("uploadOwnerUserType(%q): %v", tt.owner, err)
		}
		if got != tt.want {
			t.Fatalf("uploadOwnerUserType(%q) = %d, want %d", tt.owner, got, tt.want)
		}
	}
	if _, err := uploadOwnerUserType(upload.UploaderType("unknown")); err == nil {
		t.Fatal("unknown owner type was accepted")
	}
}
