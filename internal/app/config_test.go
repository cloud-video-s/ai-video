package app

import (
	"strings"
	"testing"
)

func TestValidateConfigRejectsExampleJWTSecretInReleaseMode(t *testing.T) {
	original := Cfg
	t.Cleanup(func() { Cfg = original })

	Cfg = Config{
		Server:   ServerConfig{Mode: "release"},
		Database: DatabaseConfig{Driver: "mysql", DBName: "app"},
		JWT:      JWTConfig{Secret: defaultJWTSecret},
		Upload: UploadConfig{
			RootDir:           "tmp",
			LocalRootDir:      "files",
			ChunkSize:         1,
			MaxBatchFiles:     1,
			SessionTTLSeconds: 1,
			ImageMaxFileSize:  1,
			VideoMaxFileSize:  1,
		},
	}

	err := validateConfig()
	if err == nil || !strings.Contains(err.Error(), "changed from the default") {
		t.Fatalf("validateConfig() error = %v, want default-secret rejection", err)
	}
}
