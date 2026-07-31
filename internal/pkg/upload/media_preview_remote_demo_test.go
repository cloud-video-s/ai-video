package upload

import (
	"context"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const (
	remoteVideoDemoURL = "https://balaaitest.oss-ap-southeast-1.aliyuncs.com/uploads/generated/1000/task-276c1375-cc84-44f9-9ff0-8e6ca355ab3a-1.mp4"
	remoteImageDemoURL = "https://balaaitest.oss-ap-southeast-1.aliyuncs.com/uploads/generated/1000/task-21-1.jpeg"
)

// TestRemoteMediaPreviewDemo generates previews from real OSS objects. It
// requires network access and the example objects to remain available.
func TestRemoteMediaPreviewDemo(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	//fmt.Println("dir:", dir)
	//return

	outputDir := dir
	options := DefaultMediaPreviewOptions()

	t.Run("image", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		outputPath := filepath.Join(outputDir, "remote-image-preview.jpg")
		if err := GenerateMediaPreview(ctx, MediaImage, remoteImageDemoURL, outputPath, options); err != nil {
			t.Fatal(err)
		}
		assertRemoteDemoJPEG(t, outputPath)
		t.Logf("remote image preview: %s", outputPath)
	})

	t.Run("video", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		outputPath := filepath.Join(outputDir, "remote-video-first-frame.png")
		if err := GenerateMediaPreview(ctx, MediaVideo, remoteVideoDemoURL, outputPath, options); err != nil {
			t.Fatal(err)
		}
		assertRemoteDemoJPEG(t, outputPath)
		t.Logf("remote video first-frame preview: %s", outputPath)
	})
}

func assertRemoteDemoJPEG(t *testing.T, path string) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	config, format, err := image.DecodeConfig(file)
	if err != nil {
		t.Fatal(err)
	}
	if format != "jpeg" || config.Width <= 0 || config.Height <= 0 {
		t.Fatalf("preview format and size = %s %dx%d", format, config.Width, config.Height)
	}
}

func TestDome(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	outputDir := dir + "/temp"
	options := DefaultMediaPreviewOptions()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	//outputPath := filepath.Join(outputDir, "222.jpg")
	//if err := GenerateMediaPreview(ctx, MediaImage, dir+"/remote-image-preview.jpg", outputPath, options); err != nil {
	//	t.Fatal(err)
	//}

	outputPath := filepath.Join(outputDir, "remote-video-first-frame.png")
	if err := GenerateMediaPreview(ctx, MediaVideo, remoteVideoDemoURL, outputPath, options); err != nil {
		t.Fatal(err)
	}
	fmt.Printf("remote image preview: %s", outputPath)
}
