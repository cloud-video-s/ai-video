package service

import "testing"

func TestValidateUploadConfigValue(t *testing.T) {
	tests := []struct {
		key     string
		value   string
		wantErr bool
	}{
		{key: "upload.image_extensions", value: ".jpg,.png"},
		{key: "upload.video_extensions", value: ".mp4,.webm"},
		{key: "upload.image_extensions", value: ".mp4", wantErr: true},
		{key: "upload.video_extensions", value: "", wantErr: true},
		{key: "upload.image_max_file_size", value: "1048576"},
		{key: "upload.video_max_file_size", value: "107374182400"},
		{key: "upload.image_max_file_size", value: "1024", wantErr: true},
		{key: "upload.video_max_file_size", value: "invalid", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.key+"/"+tt.value, func(t *testing.T) {
			err := validateConfigValue(tt.key, tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateConfigValue() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
