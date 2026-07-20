package model

import (
	"sync"
	"testing"

	"gorm.io/gorm/schema"
)

func TestVideoUserIdentitySchema(t *testing.T) {
	parsed, err := schema.Parse(&VideoUserIdentity{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Table != "video_user_identity" {
		t.Fatalf("table = %q", parsed.Table)
	}
	indexes := make(map[string]*schema.Index)
	for _, index := range parsed.ParseIndexes() {
		indexes[index.Name] = index
	}
	for _, indexName := range []string{"uk_user_identity_subject", "uk_user_identity_user_provider"} {
		index := indexes[indexName]
		if index == nil || index.Class != "UNIQUE" {
			t.Fatalf("missing unique index %s", indexName)
		}
	}
	relation := parsed.Relationships.Relations["User"]
	if relation == nil || len(relation.References) != 1 || relation.References[0].ForeignKey.DBName != "user_id" {
		t.Fatal("User relationship is not configured with user_id")
	}
}
