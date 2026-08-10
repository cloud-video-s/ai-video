package upload

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestManagerStoresAllowedImagesWithoutTranscoding(t *testing.T) {
	tests := []struct {
		name        string
		fileName    string
		contentType string
		data        []byte
	}{
		{name: "jpeg", fileName: "original.jpeg", contentType: "image/jpeg", data: append([]byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00}, bytes.Repeat([]byte{0x5a}, 32)...)},
		{name: "png", fileName: "original.png", contentType: "image/png", data: append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0xa5}, 32)...)},
		{name: "gif", fileName: "original.gif", contentType: "image/gif", data: append([]byte("GIF89a"), bytes.Repeat([]byte{0x3c}, 32)...)},
		{name: "webp", fileName: "original.webp", contentType: "image/webp", data: append([]byte("RIFF\x24\x00\x00\x00WEBPVP8 "), bytes.Repeat([]byte{0xc3}, 32)...)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			config := DefaultConfig()
			config.RootDir = root
			manager, err := NewManager(config)
			if err != nil {
				t.Fatalf("NewManager() error = %v", err)
			}

			sessions, err := manager.InitiateImages(context.Background(), []FileSpec{{
				FileName: test.fileName, Size: int64(len(test.data)), ContentType: test.contentType,
			}})
			if err != nil {
				t.Fatalf("InitiateImages() error = %v", err)
			}
			if _, err = manager.PutChunk(context.Background(), sessions[0].UploadID, 0, bytes.NewReader(test.data), ""); err != nil {
				t.Fatalf("PutChunk() error = %v", err)
			}

			completed, err := manager.Complete(context.Background(), sessions[0].UploadID)
			if err != nil {
				t.Fatalf("Complete() error = %v", err)
			}
			if completed.Extension != filepath.Ext(test.fileName) {
				t.Fatalf("Extension = %q, want %q", completed.Extension, filepath.Ext(test.fileName))
			}
			if completed.ContentType != test.contentType {
				t.Fatalf("ContentType = %q, want %q", completed.ContentType, test.contentType)
			}

			stored, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(completed.FilePath)))
			if err != nil {
				t.Fatalf("ReadFile() error = %v", err)
			}
			if !bytes.Equal(stored, test.data) {
				t.Fatal("stored image bytes differ from the uploaded original")
			}
		})
	}
}
