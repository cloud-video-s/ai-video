package model

import (
	"reflect"
	"sync"
	"testing"

	"gorm.io/gorm/schema"
)

func TestVideoBannerSchemaAssociations(t *testing.T) {
	parsed, err := schema.Parse(&VideoBanner{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Table != "video_banner" {
		t.Fatalf("table = %q", parsed.Table)
	}
	if parsed.LookUpField("position_key") != nil {
		t.Fatal("video_banner must not retain the legacy position_key column")
	}
	jumpType := parsed.FieldsByName["JumpType"]
	if jumpType == nil || jumpType.FieldType.Kind() != reflect.Uint8 || jumpType.GORMDataType != schema.Uint {
		t.Fatalf("JumpType schema = %#v, want uint8 numeric field", jumpType)
	}
	sortField := parsed.FieldsByName["Sort"]
	if sortField == nil || sortField.FieldType.Kind() != reflect.Uint64 {
		t.Fatalf("Sort schema = %#v, want uint64 field", sortField)
	}

	wantJoinTables := map[string]string{
		"DisplayPositions": "video_banner_display_position",
		"Countries":        "video_banner_country",
		"Channels":         "video_banner_channel",
		"Packages":         "video_banner_package",
	}
	for association, wantTable := range wantJoinTables {
		relation := parsed.Relationships.Relations[association]
		if relation == nil || relation.JoinTable == nil {
			t.Fatalf("association %s has no join table", association)
		}
		if relation.JoinTable.Table != wantTable {
			t.Fatalf("association %s join table = %q, want %q", association, relation.JoinTable.Table, wantTable)
		}
		if association == "DisplayPositions" {
			if relation.JoinTable.FieldsByDBName["banner_id"] == nil || relation.JoinTable.FieldsByDBName["position_key"] == nil {
				t.Fatalf("DisplayPositions join columns = %#v, want banner_id and position_key", relation.JoinTable.FieldsByDBName)
			}
			if relation.JoinTable.FieldsByDBName["display_position_id"] != nil {
				t.Fatal("DisplayPositions join table still contains display_position_id")
			}
		}
	}

	template := parsed.Relationships.Relations["Template"]
	if template == nil || len(template.References) != 1 {
		t.Fatalf("Template relationship is not configured")
	}
	if template.References[0].ForeignKey.DBName != "template_id" {
		t.Fatalf("Template foreign key = %q", template.References[0].ForeignKey.DBName)
	}
}
