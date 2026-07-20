package model

import (
	"sync"
	"testing"

	"gorm.io/gorm/schema"
)

func TestVideoPointsPackageSchema(t *testing.T) {
	parsed, err := schema.Parse(&VideoPointsPackage{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Table != "video_points_package" {
		t.Fatalf("table=%q", parsed.Table)
	}
	productIDIndex := parsed.LookIndex("ProductID")
	if productIDIndex == nil || productIDIndex.Class != "UNIQUE" {
		t.Fatal("product_id must have a database unique constraint")
	}
	relation := parsed.Relationships.Relations["Channels"]
	if relation == nil || relation.JoinTable == nil {
		t.Fatal("channels association is missing")
	}
	if relation.JoinTable.Table != "video_points_package_channel" {
		t.Fatalf("join table=%q", relation.JoinTable.Table)
	}
	for _, column := range []string{"points_package_id", "channel_id"} {
		if relation.JoinTable.LookUpField(column) == nil {
			t.Fatalf("join column %s is missing", column)
		}
	}
}
